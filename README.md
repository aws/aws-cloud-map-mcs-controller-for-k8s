# AWS Cloud Map MCS Controller for K8s

AWS Cloud Map MCS Controller for K8s is a controller that implements existing multi-cluster services API that allows services to communicate across multiple clusters. The implementation relies on [AWS Cloud Map](https://aws.amazon.com/cloud-map/) for enabling cross-cluster service discovery.

[![Deploy status](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml/badge.svg)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues)

# How to build and run

Pre-requisite: Create Private DNS Namespace in Cloud Map `demo`

Set region
```
export AWS_REGION=us-west-2
```

Spin up a local Kubernetes cluster using `kind`

```
kind create cluster
kind export kubeconfig
```

Install custom CRDs (`ServiceImport`, `ServiceExport`) to the cluster

```
make install
```

Run controller

```
make run
```

Create a testing deployment

```
# my-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: demo
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 5
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
```

Create a testing Service

```
# my-service.yaml
kind: Service
apiVersion: v1
metadata:
  namespace: demo
  name: my-service-name
spec:
  selector:
    app: nginx
  ports:
    - port: 8080
      targetPort: 80
```

Create a testing ServiceExport resource

```
# my-export.yaml

kind: ServiceExport
apiVersion: multicluster.x-k8s.io/v1alpha1
metadata:
 namespace: demo
 name: my-service-name
```

Apply config files

```
kubectl create namespace demo
kubectl apply -f my-deployment.yaml
kubectl apply -f my-service.yaml
kubectl apply -f my-export.yaml
```

Check running controller if it correctly detects newly created resource

```
2021-07-09T14:31:26.933-0700	INFO	controllers.ServiceExport	updating Cloud Map service	{"serviceexport": "demo/my-service-name", "namespace": "demo", "name": "my-service-name"}
2021-07-09T14:31:26.933-0700	INFO	cloudmap	fetching a service	{"namespaceName": "demo", "serviceName": "my-service-name"}
2021-07-09T14:31:27.341-0700	INFO	cloudmap	creating a new service	{"namespace": "demo", "name": "my-service-name"}
```

# How to generate mocks and run unit tests
```
make test
```
