#!/usr/bin/env bash

KIND_BIN='./bin/kind'
KUBECTL_BIN='./testbin/bin/kubectl'
LOGS='./integration/kind-test/testlog'
CONFIGS='./integration/kind-test/configs'
SCENARIOS='./integration/shared/scenarios'
NAMESPACE='aws-cloud-map-mcs-e2e'
SERVICE='e2e-service'
ENDPT_PORT=80
SERVICE_PORT=8080
KIND_SHORT='cloud-map-e2e'
CLUSTER='kind-cloud-map-e2e'
IMAGE='kindest/node:v1.19.16@sha256:dec41184d10deca01a08ea548197b77dc99eeacb56ff3e371af3193c86ca99f4'
EXPECTED_ENDPOINT_COUNT=5
UPDATED_ENDPOINT_COUNT=6
