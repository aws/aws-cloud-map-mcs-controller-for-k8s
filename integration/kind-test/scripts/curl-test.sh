#!/usr/bin/env bash

# Testing service consumption with dnsutils pod

deployment=$1

echo "performing curl to $SERVICE.$NAMESPACE.svc.clusterset.local"
http_code=$($KUBECTL_BIN exec deployment/$deployment --namespace "$NAMESPACE" -- curl -s -o /dev/null -w "%{http_code}" $SERVICE.$NAMESPACE.svc.clusterset.local)
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: Unable to curl $SERVICE.$NAMESPACE.svc.clusterset.local"
    exit $exit_code
fi

if [ "$http_code" -ne "200" ]; then
    echo "ERROR: curl $SERVICE.$NAMESPACE.svc.clusterset.local failed with $http_code"
    exit 1
fi

exit 0
