#!/usr/bin/env bash

# Testing service consumption with dnsutils pod

echo "verifying single-cluster service consumption..."

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
echo "performing dig for A/AAAA records..."
$KUBECTL_BIN exec $DNS_POD -n $NAMESPACE -- dig +all +ans $SERVICE.$NAMESPACE.svc.clusterset.local +short
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to dig service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

echo "performing dig for SRV records..."
$KUBECTL_BIN exec $DNS_POD -n $NAMESPACE -- dig +all +ans $SERVICE.$NAMESPACE.svc.clusterset.local. SRV +short
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to dig service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

echo "confirmed service consumption"
exit 0
