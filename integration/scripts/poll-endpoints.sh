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
  if ! addresses=$($KUBECTL_BIN describe endpoints --namespace "$NAMESPACE" | grep " Addresses: ")
  then
    # no endpoints ready
    continue
  fi

  endpts=$(echo "$addresses" | tr -s " " "$addresses" | cut -f 3 -d " ")

  endpt_count=$(echo "$endpts" | tr ',' '\n' | wc -l | xargs)
done

echo "$endpts"
exit 0