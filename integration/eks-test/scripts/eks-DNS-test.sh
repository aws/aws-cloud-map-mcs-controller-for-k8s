#!/usr/bin/env bash

# Testing service consumption with client-hello pod

source $1

if [ "$#" -ne 1 ]; then
    echo "test script expects common.sh as argument"
    exit 1
fi

echo "verifying cross-cluster service consumption..."

$KUBECTL_BIN exec $POD -n $NAMESPACE /bin/sh -- curl --version &>/dev/null

# install curl if not installed
if [ $? -eq 126 ]; then
    $KUBECTL_BIN exec $POD -n $NAMESPACE /bin/sh -- apk add curl
fi


$KUBECTL_BIN exec $POD -n $NAMESPACE /bin/sh -- curl -s $SERVICE.$NAMESPACE.svc.clusterset.local

if [ $? -ne 0 ]; then
    exit $?
fi

echo "confirmed service consumption"
exit 0

