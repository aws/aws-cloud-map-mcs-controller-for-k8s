#!/usr/bin/env bash

# Testing service consumption with client-hello pod

source $1

if [ "$#" -ne 1 ]; then
    echo "test script expects common.sh as argument"
    exit 1
fi

echo "verifying cross-cluster service consumption..."

$KUBECTL_BIN exec $POD -n $NAMESPACE /bin/sh -- curl --version &>/dev/null
exit_code=$?

# install curl if not installed
if [ "$exit_code" -eq 126 ]; then
    $KUBECTL_BIN exec $POD -n $NAMESPACE /bin/sh -- apk add curl
fi


$KUBECTL_BIN exec $POD -n $NAMESPACE /bin/sh -- curl -s $SERVICE.$NAMESPACE.svc.clusterset.local
exit_code=$?

if [ "$exit_code" -ne 0 ]; then
    exit $exit_code
fi

echo "confirmed service consumption"
exit 0

