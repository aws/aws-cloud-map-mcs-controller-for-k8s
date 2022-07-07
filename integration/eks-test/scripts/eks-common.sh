#!/usr/bin/env bash

export KIND_BIN='./bin/kind'
export KUBECTL_BIN='./testbin/bin/kubectl'
export LOGS='./integration/eks-test/testlog'
export CONFIGS='./integration/eks-test/configs'
export SCENARIOS='./integration/shared/scenarios'
export NAMESPACE='aws-cloud-map-mcs-eks-e2e'
export MCS_NAMESPACE='cloud-map-mcs-system'
export SERVICE='nginx-hello'
export CLIENT_POD='client-hello'
export ENDPT_PORT=80
export SERVICE_PORT=80 # from nginx-service.yaml
export EXPORT_CLS='cls1'
export IMPORT_CLS='cls2'
export EXPECTED_ENDPOINT_COUNT=3
export UPDATED_ENDPOINT_COUNT=4