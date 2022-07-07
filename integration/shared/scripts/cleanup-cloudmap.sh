#!/usr/bin/env bash

# Deletes all AWS Cloud Map resources used for integration test.

set -eo pipefail

go run ./integration/janitor/runner/main.go "$NAMESPACE"
