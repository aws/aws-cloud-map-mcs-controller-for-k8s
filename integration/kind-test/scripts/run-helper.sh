#!/usr/bin/env bash

# Helper to run test and passing different Service names

./integration/kind-test/scripts/run-tests.sh "e2e-service"
./integration/kind-test/scripts/run-tests.sh "e2e-headless"