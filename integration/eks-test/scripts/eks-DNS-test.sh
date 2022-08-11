#!/usr/bin/env bash

# Testing service consumption with client-hello pod

echo "verifying cross-cluster service consumption..."

# Install curl if not installed
$KUBECTL_BIN exec $CLIENT_POD -n $NAMESPACE /bin/sh -- curl --version &>/dev/null
exit_code=$?
if [ "$exit_code" -eq 126 ]; then
    echo "curl not installed, installing..."
    $KUBECTL_BIN exec $CLIENT_POD -n $NAMESPACE /bin/sh -- apk add curl
fi

# Perform an nslookup to cluster-local CoreDNS 
echo "performing nslookup..."
$KUBECTL_BIN exec -it $CLIENT_POD -n $NAMESPACE /bin/sh -- nslookup $SERVICE.$NAMESPACE.svc.clusterset.local
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to nslookup service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi
sleep 5

# Call to DNS server, if unable to reach, importing cluster is not able to properly consume service
echo "performing curl..."
$KUBECTL_BIN exec -it $CLIENT_POD -n $NAMESPACE /bin/sh -- curl $SERVICE.$NAMESPACE.svc.clusterset.local
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to reach service $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

echo "confirmed service consumption"
exit 0
