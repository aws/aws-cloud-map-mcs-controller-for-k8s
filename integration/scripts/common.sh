#!/usr/bin/env bash

KIND_BIN='./bin/kind'
KUBECTL_BIN='./testbin/bin/kubectl'
LOGS='./integration/testlog'
CONFIGS='./integration/configs'
SCENARIOS='./integration/scenarios'
NAMESPACE='aws-cloud-map-mcs-e2e'
SERVICE='e2e-service'
ENDPT_PORT=80
KIND_SHORT='cloud-map-e2e'
CLUSTER='kind-cloud-map-e2e'
IMAGE='kindest/node:v1.19.11@sha256:07db187ae84b4b7de440a73886f008cf903fcf5764ba8106a9fd5243d6f32729'
EXPECTED_ENDPOINT_COUNT=5
