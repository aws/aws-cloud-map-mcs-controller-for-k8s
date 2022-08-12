#!/usr/bin/env bash

# Helper to run test and passing different Service names

source ./integration/kind-test/scripts/common.sh

# ClusterIP service test
$KUBECTL_BIN apply -f "$CONFIGS/e2e-service.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-export.yaml"
./integration/kind-test/scripts/run-tests.sh "$CLUSTERIP_SERVICE" "ClusterSetIP"

# Remove ClusterIP service and imported service in order to prepare for next test
IMPORTED_SVC=$($KUBECTL_BIN get svc -n $NAMESPACE | grep 'imported' | awk '{print $1}')
$KUBECTL_BIN delete svc $IMPORTED_SVC -n $NAMESPACE
$KUBECTL_BIN delete svc $CLUSTERIP_SERVICE -n $NAMESPACE

# Headless service test
$KUBECTL_BIN apply -f "$CONFIGS/e2e-headless.yaml"
$KUBECTL_BIN apply -f "$CONFIGS/e2e-headless-export.yaml"
./integration/kind-test/scripts/run-tests.sh "$HEADLESS_SERVICE" "Headless"