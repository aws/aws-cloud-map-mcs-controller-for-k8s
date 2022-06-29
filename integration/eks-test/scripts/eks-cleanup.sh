#!/usr/bin/env bash

# Cleanup EKS cluster used for integration test.

source ./integration/eks-test/scripts/eks-common.sh

# Delete service and namespace from cluster 1 & 2
$KUBECTL_BIN config use-context $CLUSTER_1
$KUBECTL_BIN delete svc $SERVICE -n $NAMESPACE

for CRD in $($KUBECTL_BIN get crd -n $NAMESPACE | grep multicluster | cut -d " " -f 1 | xargs); do 
    $KUBECTL_BIN patch crd -n $NAMESPACE $CRD --type merge -p '{"metadata":{"finalizers": [null]}}'; 
done

$KUBECTL_BIN delete namespaces $NAMESPACE
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace cloud-map-mcs-system \
    --cluster $CLUSTER_1 \
    --wait

$KUBECTL_BIN config use-context $CLUSTER_2
$KUBECTL_BIN delete pod $POD -n $NAMESPACE
$KUBECTL_BIN delete namespaces $NAMESPACE
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace cloud-map-mcs-system \
    --cluster $CLUSTER_2 \
    --wait

$KUBECTL_BIN config use-context $CLUSTER_1
kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"
$KUBECTL_BIN config use-context $CLUSTER_2
kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

echo "EKS clusters cleaned!"

