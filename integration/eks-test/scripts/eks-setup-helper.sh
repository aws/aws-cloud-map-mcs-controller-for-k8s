#!/usr/bin/env bash

# Adding IAM service accounts
kubectl config use-context $1
kubectl create namespace $MCS_NAMESPACE
eksctl create iamserviceaccount \
--cluster $1 \
--namespace $MCS_NAMESPACE \
--name cloud-map-mcs-controller-manager \
--attach-policy-arn arn:aws:iam::aws:policy/AWSCloudMapFullAccess \
--override-existing-serviceaccounts \
--approve

# Installing controller
kubectl config use-context $1
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

