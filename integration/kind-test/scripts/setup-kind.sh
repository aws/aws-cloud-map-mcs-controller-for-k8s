#!/usr/bin/env bash

# Builds the AWS Cloud Map MCS Controller for K8s, provisions a Kubernetes clusters with Kind,
# installs Cloud Map CRDs and controller into the cluster and applies export and deployment configs.

set -e

source ./integration/kind-test/scripts/common.sh

./integration/kind-test/scripts/ensure-jq.sh

$KIND_BIN create cluster --name "$KIND_SHORT" --image "$IMAGE"
$KUBECTL_BIN config use-context "$CLUSTER"
$KUBECTL_BIN create namespace "$NAMESPACE"
make install

# install CoreDNS plugin
$KUBECTL_BIN apply -f "$CONFIGS/e2e-coredns-clusterrole.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-coredns-configmap.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-coredns-deployment.yaml"

# Add ClusterId and ClusterSetId
$KUBECTL_BIN apply -f "$CONFIGS/e2e-clusterId.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-clusterSetId.yaml"

# Deploy pods 
$KUBECTL_BIN apply -f "$CONFIGS/e2e-deployment.yaml"

# Add ClusterIP and Headless Services
$KUBECTL_BIN apply -f "$CONFIGS/e2e-service.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-export.yaml"

$KUBECTL_BIN apply -f "$CONFIGS/e2e-headless.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-headless-export.yaml"

exit 0
