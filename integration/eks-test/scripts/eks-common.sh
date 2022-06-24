#!/usr/bin/env bash

KIND_BIN='./bin/kind'
KUBECTL_BIN='./testbin/bin/kubectl'
LOGS='./integration/eks-test/testlog'
CONFIGS='./integration/eks-test/configs'
SCENARIOS='./integration/scenarios'
NAMESPACE='demo'
CM_NAMESPACE='cloud-map-mcs-system'
SERVICE='nginx-hello'
POD='client-hello'
ENDPT_PORT=80
SERVICE_PORT=80 # from nginx-service.yaml
CLUSTER_1='cls1'
CLUSTER_2='cls2'
CRD_IMPORT='serviceimports.multicluster.x-k8s.io'
CRD_EXPORT='serviceexports.multicluster.x-k8s.io'
EXPECTED_ENDPOINT_COUNT=3
UPDATED_ENDPOINT_COUNT=4