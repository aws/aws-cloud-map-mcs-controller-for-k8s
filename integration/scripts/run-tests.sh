#!/usr/bin/env bash

# Runs the AWS Cloud Map MCS Controller for K8s as a background process and tests services have been exported

set -eo pipefail

source ./integration/scripts/common.sh

$KUBECTL_BIN apply -f "$CONFIGS/e2e-deployment.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-service.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-export.yaml"

endpts=$(./integration/scripts/poll-endpoints.sh "$EXPECTED_ENDPOINT_COUNT")

mkdir -p "$LOGS"
./bin/manager &> "$LOGS/ctl.log" &
CTL_PID=$!

go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $ENDPT_PORT "$endpts"

kill $CTL_PID
