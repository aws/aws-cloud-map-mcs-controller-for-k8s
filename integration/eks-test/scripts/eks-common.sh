#!/usr/bin/env bash

KIND_BIN='./bin/kind'
KUBECTL_BIN='./testbin/bin/kubectl'
LOGS='./integration/eks-test/testlog'
CONFIGS='./integration/eks-test/configs'
SCENARIOS='./integration/shared/scenarios'
NAMESPACE='demo'
MCS_NAMESPACE='cloud-map-mcs-system'
SERVICE='nginx-hello'
CLIENT_POD='client-hello'
ENDPT_PORT=80
SERVICE_PORT=80 # from nginx-service.yaml
EXPORT_CLS='cls1'
IMPORT_CLS='cls2'
EXPECTED_ENDPOINT_COUNT=3
UPDATED_ENDPOINT_COUNT=4