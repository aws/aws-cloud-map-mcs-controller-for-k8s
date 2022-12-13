#!/usr/bin/env bash

# Runs the AWS Cloud Map MCS Controller for K8s as a background process and tests services have been exported

source ./integration/kind-test/scripts/common.sh
export SERVICE=$1
export SERVICE_TYPE=$2
export IP_TYPE=$3

# Deploy pods
$KUBECTL_BIN apply -f "$KIND_CONFIGS/e2e-deployment.yaml"
# Get deployment
deployment=$($KUBECTL_BIN get deployment --namespace "$NAMESPACE" -o json | jq -r '.items[0].metadata.name')

printf "\n***Testing Service: $SERVICE***\n"

$KUBECTL_BIN apply -f "$KIND_CONFIGS/$SERVICE.yaml"

if ! endpts=$(./integration/shared/scripts/poll-endpoints.sh "$EXPECTED_ENDPOINT_COUNT") ; then
  exit $?
fi

mkdir -p "$LOGS"
./bin/manager --zap-devel=true --zap-time-encoding=rfc3339 &> "$LOGS/ctl.log" &
CTL_PID=$!
echo "controller PID:$CTL_PID"

go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $CLUSTERID1 $CLUSTERSETID1 $ENDPT_PORT $SERVICE_PORT $SERVICE_TYPE $IP_TYPE "$endpts"
exit_code=$?

if [ "$exit_code" -eq 0 ] ; then
  ./integration/shared/scripts/test-import.sh "$EXPECTED_ENDPOINT_COUNT" "$endpts"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  ./integration/kind-test/scripts/dns-test.sh "$EXPECTED_ENDPOINT_COUNT"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  ./integration/kind-test/scripts/curl-test.sh "$deployment"
  exit_code=$?
fi

echo "sleeping..."
sleep 2

if [ "$exit_code" -eq 0 ] ; then
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
  go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $CLUSTERID1 $CLUSTERSETID1 $ENDPT_PORT $SERVICE_PORT $SERVICE_TYPE $IP_TYPE "$updated_endpoints"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  ./integration/shared/scripts/test-import.sh "$UPDATED_ENDPOINT_COUNT" "$updated_endpoints"
  exit_code=$?
fi

if [ "$exit_code" -eq 0 ] ; then
  ./integration/kind-test/scripts/dns-test.sh "$UPDATED_ENDPOINT_COUNT"
  exit_code=$?
fi

echo "Test Successful. Cleaning up..."

# Remove the deployment and delete service (should also delete ServiceExport)
if [ "$exit_code" -eq 0 ] ; then
  $KUBECTL_BIN delete -f "$KIND_CONFIGS/e2e-deployment.yaml"
  $KUBECTL_BIN delete Service $SERVICE -n $NAMESPACE
  # TODO: verify service export is not found
  # TODO: verify cloudmap resources are cleaned up
fi

echo "killing controller PID:$CTL_PID"
kill $CTL_PID
exit $exit_code
