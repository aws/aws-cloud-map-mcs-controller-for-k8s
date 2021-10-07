# AWS Cloud Map MCS Controller for K8s

[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg?color=success)](http://www.apache.org/licenses/LICENSE-2.0)
![GitHub issues](https://img.shields.io/github/issues-raw/aws/aws-cloud-map-mcs-controller-for-k8s?style=flat)

[![Deploy status](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml/badge.svg)](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/actions/workflows/deploy.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)](https://goreportcard.com/report/github.com/aws/aws-cloud-map-mcs-controller-for-k8s)

## Introduction
AWS Cloud Map multi-cluster service discovery for Kubernetes (K8s) is a controller that implements existing multi-cluster services API that allows services to communicate across multiple clusters. The implementation relies on [AWS Cloud Map](https://aws.amazon.com/cloud-map/) for enabling cross-cluster service discovery.

## Releases

AWS Cloud Map MCS Controller for K8s adheres to the [SemVer](https://semver.org/) specification. Each release updates the major version tag (eg. `vX`), a major/minor version tag (eg. `vX.Y`) and a major/minor/patch version tag (eg. `vX.Y.Z`). To see a full list of all releases, refer to our [Github releases page](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/releases).

We also maintain a `latest` tag, which is updated to stay in line with the `main` branch. We **do not** recommend installing this on any production cluster, as any new major versions updated on the `main` branch will introduce breaking changes.

## Contributing
`aws-cloud-map-mcs-controller-for-k8s` is an open source project. See [CONTRIBUTING](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/CONTRIBUTING.md) for details.

## License

This project is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0),
see [LICENSE](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/LICENSE) and [NOTICE](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/main/NOTICE) for more information.
