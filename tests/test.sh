#!/bin/bash

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
    ready=$(${OC} -n "${IOR_NAMESPACE}" get pod -l maistra=ior -o jsonpath='{.items..status.containerStatuses[0].ready}')

    if [ "${ready}" == "true" ]; then
      echo "OK"
      break
    fi

    if [ "${i}" == "${maxAttempts}" ]; then
      echo "Error"
      exit 1
    fi

    sleep 5
  done
}

function globalSetup() {
  deleteNamespace
  IOR_NAMESPACE="ior-$RANDOM"

  echo "Deploying IOR in namespace ${IOR_NAMESPACE}"
  sed -e "s|\${HUB}|${HUB}|g" \
      -e "s|\${TAG}|${TAG}|g" \
      -e "s|\${NAMESPACE}|${IOR_NAMESPACE}|g" \
      ${ROOT_DIR}/container/pod.yaml | ${OC} create -f-

  ${OC} label ns "${IOR_NAMESPACE}" "maistra.io/ior-test=true"
  waitForIORDeployment
}

function globalTearDown() {
  echo "Removing IOR"
  sed -e "s|\${HUB}|${HUB}|g" \
      -e "s|\${TAG}|${TAG}|g" \
      -e "s|\${NAMESPACE}|${IOR_NAMESPACE}|g" \
      ${ROOT_DIR}/container/pod.yaml | ${OC} delete -f-

  deleteNamespace
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
  jd -set <(echo ${wanted}) <(echo ${got})
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
  for file_in in ${TEST_DIR}/testdata/*.yaml; do
    setup
    testCreate "${file_in}"
    testEdit "${file_in}"
    testDelete
  done
}

globalSetup
spawnTests
