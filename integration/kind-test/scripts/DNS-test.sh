#!/usr/bin/env bash

# Testing service consumption with dnsutils pod

echo "verifying single-cluster service consumption..."

# Helper function to verify DNS results
checkDNS() {
    endpt_count=$(echo "$1" | wc -l | xargs)

    if [ "$2" = "Headless" ]; then
        if [ "$endpt_count" -ne "$3" ]; then
            echo "ERROR: Found $endpt_count endpoints, expected $3 endpoints"
            exit 1
        fi
    fi

    if [ "$2" = "ClusterSetIP" ]; then
        if [ "$endpt_count" -ne 1 ]; then
            echo "ERROR: Found $endpt_count endpoints, expected 1 endpoint"
            exit 1
        fi
    fi
}

# Add pod
$KUBECTL_BIN apply -f "$KIND_CONFIGS/e2e-client-hello.yaml"
$KUBECTL_BIN wait --for=condition=ready pod/$DNS_POD -n $NAMESPACE # wait until pod is deployed

# Install dig if not installed
$KUBECTL_BIN exec $DNS_POD -n $NAMESPACE -- dig -v &>/dev/null
exit_code=$?
if [ "$exit_code" -ne 0 ]; then
    echo "dig not installed, installing..."
    $KUBECTL_BIN exec $DNS_POD -n $NAMESPACE -- apk add --update bind-tools
fi

# Perform a dig to cluster-local CoreDNS 
# TODO: parse dig outputs for more precise verification - check specifics IPs?
echo "performing dig for A/AAAA records..."
addresses=$($KUBECTL_BIN exec $DNS_POD -n $NAMESPACE -- dig +all +ans $SERVICE.$NAMESPACE.svc.clusterset.local +short)
exit_code=$?
echo "$addresses"

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to dig service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

# verify DNS results
checkDNS "$addresses" "$SERVICE_TYPE" "$1"

echo "performing dig for SRV records..."
addresses=$($KUBECTL_BIN exec $DNS_POD -n $NAMESPACE -- dig +all +ans $SERVICE.$NAMESPACE.svc.clusterset.local. SRV +short)
exit_code=$?
echo "$addresses"

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to dig service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

# verify DNS results
checkDNS "$addresses" "$SERVICE_TYPE" "$1"

echo "confirmed service consumption"
exit 0
