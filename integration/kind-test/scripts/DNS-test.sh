#!/usr/bin/env bash

# Testing service consumption with dnsutils pod

echo "verifying cross-cluster service consumption..."

# Add DNS pod
$KUBECTL_BIN run $DNS_POD -n $NAMESPACE --image=tutum/dnsutils --command -- sleep infinity
exit_code=$?

$KUBECTL_BIN wait --for=condition=ready pod/$DNS_POD -n $NAMESPACE # wait until pod is deployed

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
