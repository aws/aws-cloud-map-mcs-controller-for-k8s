#!/usr/bin/env bash

# Cleanup EKS cluster used for integration test.

source ./integration/eks-test/scripts/eks-common.sh

# Delete service and namespace from export and import cluster
$KUBECTL_BIN config use-context $EXPORT_CLS
$KUBECTL_BIN delete svc $SERVICE -n $NAMESPACE

# Verfication to check if there are hanging ServiceExport or ServiceImport CRDs and clears the finalizers to allow cleanup process to continue
for CRD in $($KUBECTL_BIN get crd -n $NAMESPACE | grep multicluster | cut -d " " -f 1 | xargs); do 
    $KUBECTL_BIN patch crd -n $NAMESPACE $CRD --type merge -p '{"metadata":{"finalizers": [null]}}';
    $KUBECTL_BIN delete crd $CRD -n $NAMESPACE # CRD needs to be explictly deleted in order to ensure zero resources are hanging for future tests
done

$KUBECTL_BIN delete namespaces $NAMESPACE

# IAM Service Account needs to be explictly deleted, as not doing so creates hanging service accounts that cause permissions issues in future tests
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace $MCS_NAMESPACE \
    --cluster $EXPORT_CLS \
    --wait

$KUBECTL_BIN config use-context $IMPORT_CLS
$KUBECTL_BIN delete pod $CLIENT_POD -n $NAMESPACE
$KUBECTL_BIN delete namespaces $NAMESPACE
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace $MCS_NAMESPACE \
    --cluster $IMPORT_CLS \
    --wait

$KUBECTL_BIN config use-context $EXPORT_CLS
$KUBECTL_BIN delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"
$KUBECTL_BIN config use-context $IMPORT_CLS
$KUBECTL_BIN delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

echo "EKS clusters cleaned!"

# ./integration/shared/scripts/cleanup-cloudmap.sh
go run ./integration/janitor/runner/main.go "$NAMESPACE" "$CLUSTERID1" "$CLUSTERSETID1"
go run ./integration/janitor/runner/main.go "$NAMESPACE" "$CLUSTERID2" "$CLUSTERSETID1"
