#!/usr/bin/env bash

export KIND_BIN='./bin/kind'
export KUBECTL_BIN='./testbin/bin/kubectl'
export LOGS='./integration/kind-test/testlog'
export CONFIGS='./integration/kind-test/configs'
export SCENARIOS='./integration/shared/scenarios'
export NAMESPACE='aws-cloud-map-mcs-e2e'
export SERVICE='e2e-service'
export ENDPT_PORT=80
export SERVICE_PORT=8080
export KIND_SHORT='cloud-map-e2e'
export CLUSTER='kind-cloud-map-e2e'
export IMAGE='kindest/node:v1.19.16@sha256:dec41184d10deca01a08ea548197b77dc99eeacb56ff3e371af3193c86ca99f4'
export EXPECTED_ENDPOINT_COUNT=5
export UPDATED_ENDPOINT_COUNT=6
