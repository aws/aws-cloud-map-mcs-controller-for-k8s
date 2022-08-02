#!/usr/bin/env bash

# Runs the AWS Cloud Map MCS Controller for K8s as a background process and tests services have been exported

source ./integration/kind-test/scripts/common.sh

$KUBECTL_BIN apply -f "$CONFIGS/e2e-clusterId.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-clusterSetId.yaml"

$KUBECTL_BIN apply -f "$CONFIGS/e2e-deployment.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-service.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-export.yaml"

if ! endpts=$(./integration/shared/scripts/poll-endpoints.sh "$EXPECTED_ENDPOINT_COUNT") ; then
  exit $?
fi

mkdir -p "$LOGS"
./bin/manager &> "$LOGS/ctl.log" &
CTL_PID=$!
echo "controller PID:$CTL_PID"

go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $ENDPT_PORT $SERVICE_PORT "$endpts"
exit_code=$?

if [ "$exit_code" -eq 0 ] ; then
  ./integration/shared/scripts/test-import.sh "$EXPECTED_ENDPOINT_COUNT" "$endpts"
  exit_code=$?
fi

echo "sleeping..."
sleep 2

deployment=$($KUBECTL_BIN get deployment --namespace "$NAMESPACE" -o json | jq -r '.items[0].metadata.name')

echo "scaling the deployment $deployment to $UPDATED_ENDPOINT_COUNT"
$KUBECTL_BIN scale deployment/"$deployment" --replicas="$UPDATED_ENDPOINT_COUNT" --namespace "$NAMESPACE"
exit_code=$?

if [ "$exit_code" -eq 0 ] ; then
  if ! updated_endpoints=$(./integration/shared/scripts/poll-endpoints.sh "$UPDATED_ENDPOINT_COUNT") ; then
    exit $?
  fi

  go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $ENDPT_PORT $SERVICE_PORT "$updated_endpoints"
  exit_code=$?

  if [ "$exit_code" -eq 0 ] ; then
    ./integration/shared/scripts/test-import.sh "$UPDATED_ENDPOINT_COUNT" "$updated_endpoints"
    exit_code=$?
  fi
fi

echo "killing controller PID:$CTL_PID"
kill $CTL_PID
exit $exit_code
