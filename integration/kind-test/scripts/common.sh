#!/usr/bin/env bash

export KIND_BIN='./bin/kind'
export KUBECTL_BIN='kubectl'
export LOGS='./integration/kind-test/testlog'
export CONFIGS='./integration/kind-test/configs'
export SCENARIOS='./integration/shared/scenarios'
export NAMESPACE='aws-cloud-map-mcs-e2e'
export SERVICE='e2e-service'
export ENDPT_PORT=80
export SERVICE_PORT=8080
export KIND_SHORT='cloud-map-e2e'
export CLUSTER='kind-cloud-map-e2e'
export IMAGE='kindest/node:v1.20.15@sha256:a6ce604504db064c5e25921c6c0fffea64507109a1f2a512b1b562ac37d652f3'
export EXPECTED_ENDPOINT_COUNT=5
export UPDATED_ENDPOINT_COUNT=6
