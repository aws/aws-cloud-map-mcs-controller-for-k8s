#!/usr/bin/env bash

# Helper to run test and passing different Service names

source ./integration/kind-test/scripts/common.sh

# create test namespace
$KUBECTL_BIN create namespace "$NAMESPACE"

# If the IP Type env var is not set, default it to IPV4
if [[ -z "${IP_TYPE}" ]]; then
  IP_TYPE="IPV4Type"
fi

# ClusterIP service test
./integration/kind-test/scripts/run-tests.sh "$CLUSTERIP_SERVICE" "ClusterSetIP" $IP_TYPE
exit_code=$?
if [ "$exit_code" -ne 0 ] ; then
    echo "ERROR: Testing $CLUSTERIP_SERVICE failed"
    exit $exit_code
fi

sleep 5

# Headless service test
./integration/kind-test/scripts/run-tests.sh "$HEADLESS_SERVICE" "Headless" $IP_TYPE
exit_code=$?
if [ "$exit_code" -ne 0 ] ; then
    echo "ERROR: Testing $HEADLESS_SERVICE failed"
    exit $exit_code
fi
