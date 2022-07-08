#!/usr/bin/env bash

# Testing service consumption with client-hello pod

echo "verifying cross-cluster service consumption..."

kubectl exec $CLIENT_POD -n $NAMESPACE /bin/sh -- curl --version &>/dev/null
exit_code=$?

# Install curl if not installed
if [ "$exit_code" -eq 126 ]; then
    kubectl exec $CLIENT_POD -n $NAMESPACE /bin/sh -- apk add curl
fi

# Call to DNS server, if unable to reach, importing cluster is not able to properly consume service
kubectl exec $CLIENT_POD -n $NAMESPACE /bin/sh -- curl -s $SERVICE.$NAMESPACE.svc.clusterset.local
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    exit $exit_code
fi

echo "confirmed service consumption"
exit 0

