#!/usr/bin/env bash

# Delete EKS cluster used for integration test.

source ./integration/eks-test/scripts/eks-common.sh

# Delete service and namespace from cluster 1 & 2
$KUBECTL_BIN config use-context $CLUSTER_1
$KUBECTL_BIN delete svc $SERVICE -n $NAMESPACE
$KUBECTL_BIN delete namespaces $NAMESPACE
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace cloud-map-mcs-system \
    --cluster cls1 \
    --wait

$KUBECTL_BIN config use-context $CLUSTER_2
$KUBECTL_BIN delete pod $POD -n $NAMESPACE
$KUBECTL_BIN delete namespaces $NAMESPACE
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace cloud-map-mcs-system \
    --cluster cls2 \
    --wait

$KUBECTL_BIN config use-context $CLUSTER_1
kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"
$KUBECTL_BIN config use-context $CLUSTER_2
kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

