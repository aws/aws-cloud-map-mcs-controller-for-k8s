#!/usr/bin/env bash

source ./integration/eks-test/scripts/eks-common.sh

# Adding IAMService
$KUBECTL_BIN config use-context $CLUSTER_1
kubectl create namespace cloud-map-mcs-system
eksctl create iamserviceaccount \
--cluster $CLUSTER_1 \
--namespace cloud-map-mcs-system \
--name cloud-map-mcs-controller-manager \
--attach-policy-arn arn:aws:iam::aws:policy/AWSCloudMapFullAccess \
--override-existing-serviceaccounts \
--approve

$KUBECTL_BIN config use-context $CLUSTER_2
kubectl create namespace cloud-map-mcs-system
eksctl create iamserviceaccount \
--cluster $CLUSTER_2 \
--namespace cloud-map-mcs-system \
--name cloud-map-mcs-controller-manager \
--attach-policy-arn arn:aws:iam::aws:policy/AWSCloudMapFullAccess \
--override-existing-serviceaccounts \
--approve

# Installing controller
$KUBECTL_BIN config use-context $CLUSTER_1
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

$KUBECTL_BIN config use-context $CLUSTER_2
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

# Installing service
$KUBECTL_BIN config use-context $CLUSTER_1
$KUBECTL_BIN create namespace $NAMESPACE
$KUBECTL_BIN apply -f "$CONFIGS/nginx-deployment.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/nginx-service.yaml"

$KUBECTL_BIN config use-context $CLUSTER_2
$KUBECTL_BIN create namespace $NAMESPACE

# Creating service export
$KUBECTL_BIN config use-context $CLUSTER_1
$KUBECTL_BIN apply -f "$CONFIGS/nginx-serviceexport.yaml"

# Create client-hello pod
$KUBECTL_BIN config use-context $CLUSTER_2
$KUBECTL_BIN apply -f "$CONFIGS/client-hello.yaml"
sleep 15s
$KUBECTL_BIN exec $POD -n $NAMESPACE /bin/sh -- apk add curl ## install curl
