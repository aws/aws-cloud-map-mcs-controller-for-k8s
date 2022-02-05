# AWS Cloud Map MCS Controller for K8s

[![Documentation](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)
[![CodeQL](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/codeql-analysis.yml/badge.svg?branch=main)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/codeql-analysis.yml)
[![Build status](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/build.yml)
[![Deploy status](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml/badge.svg?branch=main)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml)
[![Integration status](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/integration-test.yml/badge.svg?branch=main)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/integration-test.yml)
[![codecov](https://codecov.io/gh/aws/aws-cloud-map-mcs-controller-for-k8s/branch/main/graph/badge.svg)](https://codecov.io/gh/aws/aws-cloud-map-mcs-controller-for-k8s)

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg?color=success)](http://www.apache.org/licenses/LICENSE-2.0)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues)
[![GitHub issues](https://img.shields.io/github/issues-raw/aws/aws-cloud-map-mcs-controller-for-k8s?style=flat)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues)
[![Go Report Card](https://goreportcard.com/badge/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)](https://goreportcard.com/report/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)

## Introduction
The AWS Cloud Map Multi-cluster Service Discovery Controller for Kubernetes (K8s) implements the [multi-cluster services API](https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api) specification, which allows services to communicate across multiple clusters. The implementation relies on [AWS Cloud Map](https://aws.amazon.com/cloud-map/) for enabling cross-cluster service discovery.

See the demo from AWS Container Day x KubeCon!

[![Watch the video](https://img.youtube.com/vi/3f0Tv7IiQQw/0.jpg)](https://youtu.be/3f0Tv7IiQQw?t=24458)

## Installation

Perform the following installation steps on each participating cluster.

- For multi-cluster service discovery and consumption, the controller should be installed on a minimum of 2 EKS clusters.
- Participating clusters should be provisioned into a single AWS account, within a single AWS region.

### Dependencies

#### Network

> ⚠ **The AWS Cloud Map MCS Controller for K8s provides service discovery and communication across multiple clusters, therefore implementations depend on end-end network connectivity between workloads provisioned within each participating cluster.** 

- In deployment scenarios where participating clusters are provisioned into separate VPCs, connectivity will depend on correctly configured  [VPC Peering](https://docs.aws.amazon.com/vpc/latest/peering/create-vpc-peering-connection.html), [inter-VPC routing](https://docs.aws.amazon.com/vpc/latest/peering/vpc-peering-routing.html), and Security Group configuration. The [VPC Reachability Analyzer](https://docs.aws.amazon.com/vpc/latest/reachability/getting-started.html) can be used to test and validate end-end connectivity between worker nodes within each cluster.
- Undefined behavior may occur if controllers are deployed without the required network connectivity between clusters.

#### Configure CoreDNS

Install the The CoreDNS multicluster plugin into each participating cluster. The multicluster plugin enables CoreDNS to lifecycle manage DNS records for `ServiceImport` objects.

To install the plugin, run the following commands.

```bash
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/coredns-clusterrole.yaml
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/coredns-configmap.yaml
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/coredns-deployment.yaml
```

### Install Controller

To install the latest release of the controller, run the following commands.

> **_NOTE:_** AWS region environment variable can be _optionaly_ set like `export AWS_REGION=us-west-2` Otherwise the controller will infer region in the order `AWS_REGION` environment variable, ~/.aws/config file, then EC2 metadata (for EKS environment)

```sh
kubectl apply -k "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release"
```

> 📌 See [Releases](#Releases) section for details on how to install other versions.

The controller must have sufficient IAM permissions to perform required Cloud Map operations. Grant IAM access rights `AWSCloudMapFullAccess` to the controller Service Account to enable the controller to manage Cloud Map resources.

## Usage

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
kubectl create namespace example
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/example-deployment.yaml
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/example-service.yaml
kubectl apply -f https://raw.githubusercontent.com/aws/aws-cloud-map-mcs-controller-for-k8s/main/samples/example-serviceexport.yaml
```

### Import services

In your other cluster, the controller will automatically sync services registered in AWS Cloud Map by applying the appropriate `ServiceImport`. To list them all, run
```sh
kubectl get ServiceImport -A
```

## Releases

AWS Cloud Map MCS Controller for K8s adheres to the [SemVer](https://semver.org/) specification. Each release updates the major version tag (eg. `vX`), a major/minor version tag (eg. `vX.Y`) and a major/minor/patch version tag (eg. `vX.Y.Z`). To see a full list of all releases, refer to our [Github releases page](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/releases).

> **_NOTE:_** AWS region environment variable can be _optionally_ set like `export AWS_REGION=us-west-2` Otherwise controller will infer region in the order `AWS_REGION` environment variable, ~/.aws/config file, then EC2 metadata (for EKS environment)

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

## Slack community
We have an open Slack community where users may get support with integration, discuss controller functionality and provide input on our feature roadmap. https://awsappmesh.slack.com/#k8s-mcs-controller
Join the channel with this [invite](https://join.slack.com/t/awsappmesh/shared_invite/zt-dwgbt85c-Sj_md92__quV8YADKfsQSA).

## Contributing
`aws-cloud-map-mcs-controller-for-k8s` is an open source project. See [CONTRIBUTING](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/CONTRIBUTING.md) for details.

## License

This project is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0),
see [LICENSE](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/LICENSE) and [NOTICE](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/NOTICE) for more information.
