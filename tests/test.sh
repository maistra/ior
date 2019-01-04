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

function waitFor() {
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
  waitFor
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

function checkResult() {
  local file_in="${1}"
  local file_wanted="${file_in}.out"
  local bytes_wanted=$(filterYaml "$(cat ${file_wanted})")
  local output
  local qty

  # Try 3 times with a delay of 5s between retries
  for i in $(seq 1 3); do
    output=$(${OC} -n "${ISTIO_NS}" get routes -l maistra.io/gateway-namespace="${NAMESPACE}",maistra.io/generated-by=ior -o json)
    qty=$(echo "${output}" | jq '.items | length')
    if [ "${qty}" != "0" ]; then
      break
    fi
    if [ "${i}" == "3" ]; then
      echo "FAIL"
      return 1
    fi
    sleep 5
  done

  local got=$(filterYaml "${output}")
  compareResult "${bytes_wanted}" "${got}"
}

function spawnTests() {
  echo "Starting tests"
  for file_in in ${TEST_DIR}/testdata/*.yaml; do
    setup
    ${OC} -n "${NAMESPACE}" apply -f "${file_in}"
    checkResult "${file_in}"
    tearDown
  done
}

globalSetup
spawnTests
