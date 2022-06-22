#!/usr/bin/env bash

# Runs the AWS Cloud Map MCS Controller for K8s in EKS clusters and test services has been exported from one cluster and imported from the other

source ./integration/eks-test/scripts/eks-common.sh

# Checking expected endpoints number in Cluster 1
$KUBECTL_BIN config use-context $CLUSTER_1
if ! endpts=$(./integration/scripts/poll-endpoints.sh "$EXPECTED_ENDPOINT_COUNT" ./integration/eks-test/scripts/eks-common.sh); then
    exit $?
fi

# Runner
go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $ENDPT_PORT $SERVICE_PORT "$endpts"
exit_code=$?

# Check imported endpoints
if [ "$exit_code" -eq 0 ] ; then
  $KUBECTL_BIN config use-context $CLUSTER_2
  ./integration/scripts/test-import.sh "$EXPECTED_ENDPOINT_COUNT" "$endpts" ./integration/eks-test/scripts/eks-common.sh
  exit_code=$?
  
  if [ "$exit_code" -eq 0 ] ; then
    ./integration/eks-test/scripts/eks-DNS-test.sh ./integration/eks-test/scripts/eks-common.sh
    exit_code=$?
  fi
fi

echo "sleeping..."
sleep 2s

# Scaling and verifying deployment
$KUBECTL_BIN config use-context $CLUSTER_1
deployment=$($KUBECTL_BIN get deployment --namespace "$NAMESPACE" -o json | jq -r '.items[0].metadata.name')

echo "scaling the deployment $deployment to $UPDATED_ENDPOINT_COUNT"
$KUBECTL_BIN scale deployment/"$deployment" --replicas="$UPDATED_ENDPOINT_COUNT" --namespace "$NAMESPACE"
exit_code=$?

if [ "$exit_code" -eq 0 ] ; then
  if ! updated_endpoints=$(./integration/scripts/poll-endpoints.sh "$UPDATED_ENDPOINT_COUNT" ./integration/eks-test/scripts/eks-common.sh) ; then
    exit $?
  fi

  go run $SCENARIOS/runner/main.go $NAMESPACE $SERVICE $ENDPT_PORT $SERVICE_PORT "$updated_endpoints"
  exit_code=$?

  if [ "$exit_code" -eq 0 ] ; then
    $KUBECTL_BIN config use-context $CLUSTER_2
    ./integration/scripts/test-import.sh "$UPDATED_ENDPOINT_COUNT" "$updated_endpoints" ./integration/eks-test/scripts/eks-common.sh
    exit_code=$?
    
    if [ "$exit_code" -eq 0 ] ; then
      ./integration/eks-test/scripts/eks-DNS-test.sh ./integration/eks-test/scripts/eks-common.sh
      exit_code=$?
    fi
  fi
fi

# dump logs
$KUBECTL_BIN config use-context $CLUSTER_1
$KUBECTL_BIN logs -l control-plane=controller-manager -c manager --namespace cloud-map-mcs-system &> "$LOGS/ctl-1.log" 
$KUBECTL_BIN config use-context $CLUSTER_2
$KUBECTL_BIN logs -l control-plane=controller-manager -c manager --namespace cloud-map-mcs-system &> "$LOGS/ctl-2.log" 
echo "dumped logs"

