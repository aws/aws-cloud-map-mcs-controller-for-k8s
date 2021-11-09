#!/usr/bin/env bash

# Deletes Kind cluster used for integration test.

set -eo pipefail
source ./integration/scripts/common.sh

$KIND_BIN delete cluster --name "$KIND_SHORT"