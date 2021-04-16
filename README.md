# How to build and run

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

Create a testing ServiceExport resource

```
# my-export.yaml

kind: ServiceExport
apiVersion: multicluster.k8s.aws/v1alpha1
metadata:
 namespace: demo
 name: my-service-name
```

Apply config file

```
kubectl create namespace demo
kubectl apply -f my-export.yaml
```

Check running controller if it correctly detects newly created resource

```
2021-04-16T14:41:01.686-0700	INFO	controllers.ServiceExport	ServiceExport detected: demo/my-service-name
```