#!/usr/bin/env bash

# Runs the AWS Cloud Map MCS Controller for K8s in EKS clusters and test services has been exported from one cluster and imported from the other

source ./integration/eks-test/scripts/eks-common.sh

# Checking expected endpoints number in exporting cluster
$KUBECTL_BIN config use-context $EXPORT_CLS
if ! endpts=$(./integration/shared/scripts/poll-endpoints.sh "$EXPECTED_ENDPOINT_COUNT"); then
    exit $?
fi

# Runner to verify expected endpoints are exported to Cloud Map
go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $CLUSTERID1 $CLUSTERSETID1 $ENDPT_PORT $SERVICE_PORT $SERVICE_TYPE "$endpts"
exit_code=$?

# Check imported endpoints in importing cluster
if [ "$exit_code" -eq 0 ] ; then
  $KUBECTL_BIN config use-context $IMPORT_CLS
  ./integration/shared/scripts/test-import.sh "$EXPECTED_ENDPOINT_COUNT" "$endpts"
  exit_code=$?
fi
  
# Verifying that importing cluster is properly consuming services
if [ "$exit_code" -eq 0 ] ; then
  ./integration/eks-test/scripts/eks-DNS-test.sh
  exit_code=$?
fi

echo "sleeping..."
sleep 2

# Scaling and verifying deployment
if [ "$exit_code" -eq 0 ] ; then
  $KUBECTL_BIN config use-context $EXPORT_CLS
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
  $KUBECTL_BIN config use-context $IMPORT_CLS
  ./integration/shared/scripts/test-import.sh "$UPDATED_ENDPOINT_COUNT" "$updated_endpoints"
  exit_code=$?
fi
  
if [ "$exit_code" -eq 0 ] ; then
  ./integration/eks-test/scripts/eks-DNS-test.sh
  exit_code=$?
fi


# Dump logs
mkdir -p "$LOGS"
$KUBECTL_BIN config use-context $EXPORT_CLS
$KUBECTL_BIN logs -l control-plane=controller-manager -c manager --namespace $MCS_NAMESPACE &> "$LOGS/ctl-1.log" 
$KUBECTL_BIN config use-context $IMPORT_CLS
$KUBECTL_BIN logs -l control-plane=controller-manager -c manager --namespace $MCS_NAMESPACE &> "$LOGS/ctl-2.log" 
echo "dumped logs"

exit $exit_code
