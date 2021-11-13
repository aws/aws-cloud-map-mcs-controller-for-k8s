#!/usr/bin/env bash

# Test service imports were created during e2e test

set -e

source ./integration/scripts/common.sh

if [ "$#" -ne 1 ]; then
    echo "test script expects endpoint IP list as single argument"
    exit 1
fi

endpts=$1
echo "checking service imports..."

imports=$($KUBECTL_BIN get endpointslices -o json --namespace $NAMESPACE | \
  jq '.items[] | select(.metadata.ownerReferences[].name | startswith("imported")) | .endpoints[].addresses[0]')
import_count=$(echo "$imports" | wc -l | xargs)

if ((import_count != EXPECTED_ENDPOINT_COUNT)) ; then
  echo "expected $EXPECTED_ENDPOINT_COUNT imports but found $import_count"
  exit 1
fi

echo "$imports" | tr -d '"' | while read -r import; do
  echo "checking import: $import"
  if ! echo "$endpts" | grep -q "$import" ; then
    echo "exported endpoint not found: $import"
    exit 1
  fi
done

if [ $? -ne 0 ]; then
    exit $?
fi

echo "matched all imports to exported endpoints"
exit 0
