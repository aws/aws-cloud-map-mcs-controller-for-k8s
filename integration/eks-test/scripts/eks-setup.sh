#!/usr/bin/env bash

source ./integration/eks-test/scripts/eks-common.sh

# Call helper for service account and controller installation
./integration/eks-test/scripts/eks-setup-helper.sh $EXPORT_CLS
./integration/eks-test/scripts/eks-setup-helper.sh $IMPORT_CLS

# Apply ClusterProperties
$KUBECTL_BIN config use-context $EXPORT_CLS
$KUBECTL_BIN apply -f "$CONFIGS/e2e-clusterproperty-1.yaml"

$KUBECTL_BIN config use-context $IMPORT_CLS
$KUBECTL_BIN apply -f "$CONFIGS/e2e-clusterproperty-2.yaml"

# Installing service
$KUBECTL_BIN config use-context $EXPORT_CLS
$KUBECTL_BIN create namespace $NAMESPACE
$KUBECTL_BIN apply -f "$CONFIGS/nginx-deployment.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/nginx-service.yaml"

$KUBECTL_BIN config use-context $IMPORT_CLS
$KUBECTL_BIN create namespace $NAMESPACE

# Creating service export
$KUBECTL_BIN config use-context $EXPORT_CLS
$KUBECTL_BIN apply -f "$CONFIGS/nginx-serviceexport.yaml"

# Create client-hello pod
$KUBECTL_BIN config use-context $IMPORT_CLS
$KUBECTL_BIN apply -f "$CONFIGS/client-hello.yaml"
sleep 15
