#!/usr/bin/env bash

# Helper function to verify DNS results
checkDNS() {
    dns_addresses_count=$(echo "$1" | wc -l | xargs)

    if [ "$SERVICE_TYPE" = "Headless" ]; then
        if [ "$dns_addresses_count" -ne "$expected_endpoint_count" ]; then
            echo "ERROR: Found $dns_addresses_count endpoints, expected $expected_endpoint_count endpoints"
            exit 1
        fi
    fi

    if [ "$SERVICE_TYPE" = "ClusterSetIP" ]; then
        if [ "$dns_addresses_count" -ne 1 ]; then
            echo "ERROR: Found $dns_addresses_count endpoints, expected 1 endpoint"
            exit 1
        fi
    fi
}

# Testing service consumption with dnsutils pod

echo "verifying dns resolution..."

expected_endpoint_count=$1

# Install dnsutils pod
$KUBECTL_BIN apply -f "$KIND_CONFIGS/dnsutils-pod.yaml"
$KUBECTL_BIN wait --for=condition=ready pod/dnsutils # wait until pod is deployed

# Perform a dig to cluster-local CoreDNS
# TODO: parse dig outputs for more precise verification - check specifics IPs?
if [[ $IP_TYPE == "IPV4Type" ]]; then
    echo "performing dig for A/AAAA records for IPV4..."
    addresses=$($KUBECTL_BIN exec dnsutils -- dig +all +ans $SERVICE.$NAMESPACE.svc.clusterset.local +short)
    exit_code=$?
    echo "$addresses"
elif [[ $IP_TYPE == "IPV6Type" ]]; then
    echo "performing dig for A/AAAA records for IPV6..."
    addresses=$($KUBECTL_BIN exec dnsutils -- dig AAAA +all +ans $SERVICE.$NAMESPACE.svc.clusterset.local +short)
    exit_code=$?
    echo "$addresses"
else
    echo "IP_TYPE invalid"
fi

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to dig service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

# verify DNS results
checkDNS "$addresses"

echo "performing dig for SRV records..."
addresses=$($KUBECTL_BIN exec dnsutils -- dig +all +ans $SERVICE.$NAMESPACE.svc.clusterset.local. SRV +short)
exit_code=$?
echo "$addresses"

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to dig service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

# verify DNS results
checkDNS "$addresses"

echo "confirmed dns resolution"
exit 0
