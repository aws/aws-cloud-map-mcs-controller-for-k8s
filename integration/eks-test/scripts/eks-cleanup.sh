#!/usr/bin/env bash

# Delete EKS cluster used for integration test.

source ./integration/eks-test/scripts/eks-common.sh

# Delete service and namespace from cluster 1 & 2
$KUBECTL_BIN config use-context $CLUSTER_1
$KUBECTL_BIN delete svc $SERVICE -n $NAMESPACE
sleep 30s
$KUBECTL_BIN delete namespaces $NAMESPACE
sleep 60s

$KUBECTL_BIN config use-context $CLUSTER_2
$KUBECTL_BIN delete pod $POD -n $NAMESPACE
$KUBECTL_BIN delete namespaces $NAMESPACE

# $KUBECTL_BIN config use-context $CLUSTER_1
# kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"
# $KUBECTL_BIN config use-context $CLUSTER_2
# kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

