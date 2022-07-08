#!/usr/bin/env bash

# Cleanup EKS cluster used for integration test.

source ./integration/eks-test/scripts/eks-common.sh

# Delete service and namespace from export and import cluster
kubectl config use-context $EXPORT_CLS
kubectl delete svc $SERVICE -n $NAMESPACE

# Verfication to check if there are hanging ServiceExport or ServiceImport CRDs and clears the finalizers to allow cleanup process to continue
for CRD in $(kubectl get crd -n $NAMESPACE | grep multicluster | cut -d " " -f 1 | xargs); do 
    kubectl patch crd -n $NAMESPACE $CRD --type merge -p '{"metadata":{"finalizers": [null]}}';
    kubectl delete crd $CRD -n $NAMESPACE # CRD needs to be explictly deleted in order to ensure zero resources are hanging for future tests
done

kubectl delete namespaces $NAMESPACE

# IAM Service Account needs to be explictly deleted, as not doing so creates hanging service accounts that cause permissions issues in future tests
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace $MCS_NAMESPACE \
    --cluster $EXPORT_CLS \
    --wait

kubectl config use-context $IMPORT_CLS
kubectl delete pod $CLIENT_POD -n $NAMESPACE
kubectl delete namespaces $NAMESPACE
eksctl delete iamserviceaccount \
    --name cloud-map-mcs-controller-manager \
    --namespace $MCS_NAMESPACE \
    --cluster $IMPORT_CLS \
    --wait

kubectl config use-context $EXPORT_CLS
kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"
kubectl config use-context $IMPORT_CLS
kubectl delete -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"

echo "EKS clusters cleaned!"

./integration/shared/scripts/cleanup-cloudmap.sh

