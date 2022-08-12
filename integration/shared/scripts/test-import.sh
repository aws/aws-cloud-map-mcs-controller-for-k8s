#!/usr/bin/env bash

# Test service imports were created during e2e test

set -e

expected_endpoint_count=$1
endpoints=$2
echo "checking service imports..."

import_count=0
poll_count=0
while ((import_count < expected_endpoint_count))
do
  sleep 1
  if ((poll_count++ > 30)) ; then
    echo "timed out polling for import endpoints"
    exit 1
  fi

  imports=$($KUBECTL_BIN get endpointslices -o json --namespace $NAMESPACE | \
    jq '.items[] | select(.metadata.ownerReferences[].name | startswith("imported")) | .endpoints[].addresses[0]')
  echo "import endpoint list from kubectl:"
  echo "$imports"

  import_count=$(echo "$imports" | wc -l | xargs)
done

echo "$imports" | tr -d '"' | while read -r import; do
  echo "checking import: $import"
  if ! echo "$endpoints" | grep -q "$import" ; then
    echo "exported endpoint not found: $import"
    exit 1
  fi
done

if [ $? -ne 0 ]; then
    exit $?
fi

echo "matched all imports to exported endpoints"
exit 0
