#!/bin/bash

TEST_DIR="$(cd "$(dirname "${0}")" && pwd -P)"
OC=${OC:-oc}
ISTIO_NS=${ISTIO_NS:-"istio-system"}
NAMESPACE=""

function deleteNamespace() {
  ${OC} delete --ignore-not-found=true --now=true namespace "${1}"
}

function setup() {
  NAMESPACE="ior-test-$RANDOM"

  deleteNamespace "${NAMESPACE}"
  ${OC} create ns "${NAMESPACE}"
  ${OC} label ns "${NAMESPACE}" "maistra.io/ior-test=true"
}

function tearDown() {
  deleteNamespace "${NAMESPACE}"
}
trap tearDown EXIT

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
  for file_in in ${TEST_DIR}/testdata/*.yaml; do
    setup
    ${OC} -n "${NAMESPACE}" apply -f "${file_in}"
    checkResult "${file_in}"
    tearDown
  done
}

spawnTests
