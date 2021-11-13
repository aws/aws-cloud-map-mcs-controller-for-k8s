#!/usr/bin/env bash

# Poll for endpoints to become active

set -e
source ./integration/scripts/common.sh

endpt_count=0
poll_count=0
while ((endpt_count < $1))
do
  if ((poll_count++ > 30)) ; then
    echo "timed out polling for endpoints"
    exit 1
  fi

  sleep 2s
  if ! addresses=$($KUBECTL_BIN get endpointslices -o json --namespace "$NAMESPACE" | \
    jq '.items[] | select(.metadata.ownerReferences[].name=="e2e-service") | .endpoints[].addresses[0]' 2> /dev/null)
  then
    # no endpoints ready
    continue
  fi

  endpt_count=$(echo "$addresses" | wc -l | xargs)
done

echo "$addresses" | tr -d '"' | paste -sd "," -
exit 0
