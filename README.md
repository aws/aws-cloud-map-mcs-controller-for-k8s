# AWS Cloud Map MCS Controller for K8s

[![Documentation](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)
[![CodeQL](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/codeql-analysis.yml/badge.svg?branch=main)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/codeql-analysis.yml)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues)
[![Build status](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/build.yml)
[![Deploy status](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml/badge.svg?branch=main)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml)

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg?color=success)](http://www.apache.org/licenses/LICENSE-2.0)
[![GitHub issues](https://img.shields.io/github/issues-raw/aws/aws-cloud-map-mcs-controller-for-k8s?style=flat)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues)
[![Go Report Card](https://goreportcard.com/badge/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)](https://goreportcard.com/report/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)

## Introduction
AWS Cloud Map multi-cluster service discovery for Kubernetes (K8s) is a controller that implements existing multi-cluster services API that allows services to communicate across multiple clusters. The implementation relies on [AWS Cloud Map](https://aws.amazon.com/cloud-map/) for enabling cross-cluster service discovery.

## Usage
> ⚠ **There must exist network connectivity (i.e. VPC peering, security group rules, ACLs, etc.) between clusters**: Undefined behavior may occur if controller is set up without network connectivity between clusters.

### Setup clusters

First, install the controller with latest release on at least 2 AWS EKS clusters. Nodes must have sufficient IAM permissions to perform CloudMap operations.

```sh
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release"
```

> 📌 See [Releases](#Releases) section for details on how to install other versions.

### Export services

Then assuming you already have a Service installed, apply a `ServiceExport` yaml to the cluster in which you want to export a service. This can be done for each service you want to export.

```yaml
kind: ServiceExport
apiVersion: multicluster.x-k8s.io/v1alpha1
metadata:
  namespace: [Your service namespace here]
  name: [Your service name]
```

**Example:** This will export a service with name *my-amazing-service* in namespace *hello*
```yaml
kind: ServiceExport
apiVersion: multicluster.x-k8s.io/v1alpha1
metadata:
  namespace: hello
  name: my-amazing-service
```

*See the `samples` directory for a set of example yaml files to set up a service and export it. To apply the sample files run*
```sh
kubectl create namespace demo
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/demo-deployment.yaml
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/demo-service.yaml
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/demo-export.yaml
```

### Import services

In your other cluster, the controller will automatically sync services registered in AWS CloudMap by applying the appropriate `ServiceImport`. To list them all, run
```sh
kubectl get ServiceImport -A
```

## Releases

AWS Cloud Map MCS Controller for K8s adheres to the [SemVer](https://semver.org/) specification. Each release updates the major version tag (eg. `vX`), a major/minor version tag (eg. `vX.Y`) and a major/minor/patch version tag (eg. `vX.Y.Z`). To see a full list of all releases, refer to our [Github releases page](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/releases).

To install from a release run
```sh
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release[?ref=*git version tag*]"
```

Example to install latest release
```sh
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release"
```

Example to install v0.1.0
```sh
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release?ref=v0.1.0"
```

We also maintain a `latest` tag, which is updated to stay in line with the `main` branch. We **do not** recommend installing this on any production cluster, as any new major versions updated on the `main` branch will introduce breaking changes.

To install from `latest` tag run
```sh
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_latest"
```

## Contributing
`aws-cloud-map-mcs-controller-for-k8s` is an open source project. See [CONTRIBUTING](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/CONTRIBUTING.md) for details.

## License

This project is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0),
see [LICENSE](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/LICENSE) and [NOTICE](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/NOTICE) for more information.
