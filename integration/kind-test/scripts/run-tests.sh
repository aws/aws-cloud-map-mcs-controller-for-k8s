#!/usr/bin/env bash

# Runs the AWS Cloud Map MCS Controller for K8s as a background process and tests services have been exported

source ./integration/kind-test/scripts/common.sh
export SERVICE=$1
export SERVICE_TYPE=$2

echo "testing service: $SERVICE"

if ! endpts=$(./integration/shared/scripts/poll-endpoints.sh "$EXPECTED_ENDPOINT_COUNT") ; then
  exit $?
fi

mkdir -p "$LOGS"
./bin/manager &> "$LOGS/ctl.log" &
CTL_PID=$!
echo "controller PID:$CTL_PID"

go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $CLUSTERID1 $CLUSTERSETID1 $ENDPT_PORT $SERVICE_PORT $SERVICE_TYPE "$endpts" 
exit_code=$?

if [ "$exit_code" -eq 0 ] ; then
  ./integration/shared/scripts/test-import.sh "$EXPECTED_ENDPOINT_COUNT" "$endpts"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  ./integration/kind-test/scripts/DNS-test.sh "$EXPECTED_ENDPOINT_COUNT"
  exit_code=$?
fi

echo "sleeping..."
sleep 2

if [ "$exit_code" -eq 0 ] ; then
  deployment=$($KUBECTL_BIN get deployment --namespace "$NAMESPACE" -o json | jq -r '.items[0].metadata.name')

  echo "scaling the deployment $deployment to $UPDATED_ENDPOINT_COUNT"
  $KUBECTL_BIN scale deployment/"$deployment" --replicas="$UPDATED_ENDPOINT_COUNT" --namespace "$NAMESPACE"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  if ! updated_endpoints=$(./integration/shared/scripts/poll-endpoints.sh "$UPDATED_ENDPOINT_COUNT") ; then
    exit $?
  fi
fi

if [ "$exit_code" -eq 0 ] ; then
  go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $CLUSTERID1 $CLUSTERSETID1 $ENDPT_PORT $SERVICE_PORT $SERVICE_TYPE "$updated_endpoints"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  ./integration/shared/scripts/test-import.sh "$UPDATED_ENDPOINT_COUNT" "$updated_endpoints"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  ./integration/kind-test/scripts/DNS-test.sh "$UPDATED_ENDPOINT_COUNT"
  exit_code=$?
fi

# Scale deployment back down for future test and delete service export
if [ "$exit_code" -eq 0 ] ; then
  $KUBECTL_BIN scale deployment/"$deployment" --replicas="$EXPECTED_ENDPOINT_COUNT" --namespace "$NAMESPACE"
  $KUBECTL_BIN delete ServiceExport $SERVICE -n $NAMESPACE
  sleep 5
fi

echo "killing controller PID:$CTL_PID"
kill $CTL_PID
exit $exit_code
