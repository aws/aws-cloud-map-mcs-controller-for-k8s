#!/usr/bin/env bash

# Builds the AWS Cloud Map MCS Controller for K8s, provisions a Kubernetes clusters with Kind,
# installs Cloud Map CRDs and controller into the cluster and applies export and deployment configs.

set -e

source ./integration/kind-test/scripts/common.sh

./integration/kind-test/scripts/ensure-jq.sh

$KIND_BIN create cluster --name "$KIND_SHORT" --image "$IMAGE" --config=./integration/kind-test/configs/ipv6.yaml
$KUBECTL_BIN config use-context "$CLUSTER"
make install

# Install CoreDNS plugin
$KUBECTL_BIN apply -f "$SHARED_CONFIGS/coredns-clusterrole.yaml"
$KUBECTL_BIN apply -f "$SHARED_CONFIGS/coredns-configmap.yaml"
$KUBECTL_BIN apply -f "$KIND_CONFIGS/coredns-deployment.yaml"

# Add ClusterId and ClusterSetId
$KUBECTL_BIN apply -f "$KIND_CONFIGS/e2e-clusterproperty.yaml"
