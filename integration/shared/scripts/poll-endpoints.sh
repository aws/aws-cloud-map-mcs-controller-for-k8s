#!/usr/bin/env bash

# Poll for endpoints to become active

set -e

endpt_count=0
poll_count=0
while ((endpt_count < $1))
do
  if ((poll_count++ > 30)) ; then
    echo "timed out polling for endpoints" >&2
    exit 1
  fi

  sleep 2s
  if ! addresses=$($KUBECTL_BIN get endpointslices -o json --namespace "$NAMESPACE" | \
    jq --arg SERVICE "$SERVICE" '.items[] | select(.metadata.ownerReferences[].name==$SERVICE) | .endpoints[].addresses[0]' 2> /dev/null)
  then
    # no endpoints ready
    continue
  fi

  endpt_count=$(echo "$addresses" | wc -l | xargs)
done

echo "$addresses" | tr -d '"' | paste -sd "," - 
echo "matched number of endpoints to expected count" >&2
exit 0
