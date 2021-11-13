#!/usr/bin/env bash

# Runs the AWS Cloud Map MCS Controller for K8s as a background process and tests services have been exported

source ./integration/scripts/common.sh

$KUBECTL_BIN apply -f "$CONFIGS/e2e-deployment.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-service.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-export.yaml"

if ! endpts=$(./integration/scripts/poll-endpoints.sh "$EXPECTED_ENDPOINT_COUNT") ; then
  exit $?
fi

mkdir -p "$LOGS"
./bin/manager &> "$LOGS/ctl.log" &
CTL_PID=$!
echo "controller PID:$CTL_PID"

go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $ENDPT_PORT "$endpts"
exit_code=$?

if [ "$exit_code" -eq 0 ] ; then
  ./integration/scripts/test-import.sh "$endpts"
  exit_code=$?
fi

echo "killing controller PID:$CTL_PID"
kill $CTL_PID
exit $exit_code
