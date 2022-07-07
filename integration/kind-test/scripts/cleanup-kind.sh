#!/usr/bin/env bash

# Deletes Kind cluster used for integration test.

set -eo pipefail
source ./integration/kind-test/scripts/common.sh

$KIND_BIN delete cluster --name "$KIND_SHORT"

./integration/shared/scripts/cleanup-cloudmap.sh
