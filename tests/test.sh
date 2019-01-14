#!/bin/bash

## Copyright 2019 Red Hat, Inc.
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.

TEST_DIR="$(cd "$(dirname "${0}")" && pwd -P)"
ROOT_DIR="$(cd "$(dirname "${0}")/.." && pwd -P)"
OC=${OC:-oc}
ISTIO_NS=${ISTIO_NS:-"istio-system"}
NAMESPACE=""
HUB=${HUB:-"docker.io/maistra"}
TAG=${TAG:-"latest"}

function deleteNamespace() {
  if [ -z "${1}" ]; then
    ${OC} delete --ignore-not-found=true --now=true namespace -l "maistra.io/ior-test=true"
    return
  fi

  ${OC} delete --ignore-not-found=true --now=true namespace "${1}"
}

function waitForIORDeployment() {
  local maxAttempts="10"
  local ready

  echo -n "Waiting for IOR deployment to complete... "

  for i in $(seq 1 ${maxAttempts}); do
    # Sleep at the beginning as it's unlikely it was processed so quickly
    sleep 5

    ready=$(${OC} -n "${IOR_NAMESPACE}" get pod -l maistra=ior -o jsonpath='{.items..status.containerStatuses[0].ready}')

    if [ "${ready}" == "true" ]; then
      echo "OK"
      return 0
    fi
  done

  echo "FAIL"
  return 1
}

function timeSpent() {
  local now=$(date +%s)
  local seconds=$((now-START_TIME))

  echo "(${seconds}.00s)"
}

function startTest() {
  TESTNAME="${1:-${FUNCNAME[1]}}"
  echo "=== RUN   ${TESTNAME}"
  START_TIME=$(date +%s)
}

function passTest() {
  echo "--- PASS: ${TESTNAME} $(timeSpent)"
}

function failTest() {
  FAIL=1
  echo "--- FAIL: ${TESTNAME} $(timeSpent)"
}

function run() {
  $*

  if [ $? -ne 0 ]; then
    failTest
    return 1
  fi

  return 0
}

function globalSetup() {
  startTest

  deleteNamespace
  IOR_NAMESPACE="ior-$RANDOM"

  echo "Deploying IOR in namespace ${IOR_NAMESPACE}"
  run sed -e "s|\${HUB}|${HUB}|g" \
      -e "s|\${TAG}|${TAG}|g" \
      -e "s|\${NAMESPACE}|${IOR_NAMESPACE}|g" \
      ${ROOT_DIR}/container/pod.yaml | ${OC} create -f- || return 1

  run ${OC} label ns "${IOR_NAMESPACE}" "maistra.io/ior-test=true" || return 1
  run waitForIORDeployment || return 1

  passTest
}

function globalTearDown() {
  if [ -n "${GLOBAL_TEAR_DOWN}" ]; then
    return
  fi
  GLOBAL_TEAR_DOWN=1

  startTest

  echo "Removing IOR"
  run sed -e "s|\${HUB}|${HUB}|g" \
      -e "s|\${TAG}|${TAG}|g" \
      -e "s|\${NAMESPACE}|${IOR_NAMESPACE}|g" \
      ${ROOT_DIR}/container/pod.yaml | ${OC} delete -f- || return 1

  run deleteNamespace || return 1

  passTest
}
trap globalTearDown EXIT

function setup() {
  NAMESPACE="ior-test-$RANDOM"

  deleteNamespace "${NAMESPACE}"
  ${OC} create ns "${NAMESPACE}"
  ${OC} label ns "${NAMESPACE}" "maistra.io/ior-test=true"
}

function tearDown() {
  deleteNamespace "${NAMESPACE}"
}

function compareResult() {
  local wanted="${1}"
  local got="${2}"

  #echo ${wanted} | jq -S . > /tmp/wanted.json
  #echo ${got} | jq -S . > /tmp/got.json

  #diff <(echo ${wanted} | jq -S .) <(echo ${got} | jq -S .)
  local output=$(jd -set <(echo ${wanted}) <(echo ${got}))
  if [ -z "${output}" ]; then
    return
  fi

  echo "${output}"
  return 1
}

function filterYaml() {
  echo "${1}" | jq -S -f ${TEST_DIR}/filters.jq
}

function waitForIORProcessing() {
  local maxAttempts="5"
  local isDeletion="${1:-false}"
  local output
  local op

  if [ "${isDeletion}" = "true" ]; then
    op="="
  else
    op="!="
  fi

  for i in $(seq 1 ${maxAttempts}); do
    # Sleep at the beginning as it's unlikely it was processed so quickly
    sleep 5

    output=$(${OC} -n "${ISTIO_NS}" get routes -l maistra.io/gateway-namespace="${NAMESPACE}",maistra.io/generated-by=ior -o json)
    qty=$(echo "${output}" | jq '.items | length')

    if [ "${qty}" ${op} "0" ]; then

      if [ "${isDeletion}" != "true" ]; then
        echo "${output}"
      fi

      return 0
    fi

    if [ "${i}" == "${maxAttempts}" ]; then
      echo "FAIL"
      return 1
    fi
  done
}

function checkResult() {
  local file_in="${1}"
  local file_wanted="${file_in}.out"
  local bytes_wanted=$(filterYaml "$(cat ${file_wanted})")

  local output=$(waitForIORProcessing false)
  local got=$(filterYaml "${output}")
  compareResult "${bytes_wanted}" "${got}"
}

function testCreate() {
  local file_in="${1}"

  ${OC} -n "${NAMESPACE}" apply -f "${file_in}"
  checkResult "${file_in}"
}

function testEdit() {
  local file_in="${1}"
  local file_edited_in="${file_in}.edited"

  if [ ! -f "${file_edited_in}" ]; then
    return
  fi

  ${OC} -n "${NAMESPACE}" apply -f "${file_edited_in}"
  checkResult "${file_edited_in}"
}

function testDelete() {
  tearDown
  waitForIORProcessing true
}

function spawnTests() {
  echo "Starting tests"
  local testName

  for file_in in ${TEST_DIR}/testdata/*.yaml; do
    testName="Testing file: ${file_in}"
    startTest "${testName}"

    run setup || continue
    run testCreate "${file_in}" || continue
    run testEdit "${file_in}" || continue
    run testDelete || continue

    passTest
  done
}

function printResults() {
  if [ -n "${FAIL}" ]; then
    echo "FAIL"
  else
    echo "PASS"
  fi
}

globalSetup
spawnTests
globalTearDown
printResults
