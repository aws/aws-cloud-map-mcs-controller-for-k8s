# Contributing Guidelines

Thank you for your interest in contributing to our project. Whether it's a bug report, new feature, correction, or additional
documentation, we greatly value feedback and contributions from our community.

Please read through this document before submitting any issues or pull requests to ensure we have all the necessary
information to effectively respond to your bug report or contribution.

<!-- TOC -->
* [Contributing Guidelines](#contributing-guidelines)
  * [Architecture Overview](#architecture-overview)
  * [Getting Started](#getting-started)
    * [Build and Unit Tests](#build-and-unit-tests)
    * [Local Setup](#local-setup)
      * [Prerequisites](#prerequisites)
      * [Kind Cluster Setup](#kind-cluster-setup)
      * [Run the controller from outside the cluster](#run-the-controller-from-outside-the-cluster)
      * [Build and deploy controller into the cluster](#build-and-deploy-controller-into-the-cluster)
    * [Local integration testing](#local-integration-testing)
  * [Build and push docker image](#build-and-push-docker-image)
  * [Reporting Bugs/Feature Requests](#reporting-bugsfeature-requests)
  * [Contributing via Pull Requests](#contributing-via-pull-requests)
  * [Finding contributions to work on](#finding-contributions-to-work-on)
  * [Code of Conduct](#code-of-conduct)
  * [Security issue notifications](#security-issue-notifications)
  * [Licensing](#licensing)
<!-- TOC -->

## Architecture Overview

![Architecture diagram](docs/architecture-overview.png?raw=true)

* `pkg/controllers/serviceexport_controller` is watching changes on K8s `ServiceExport` resources (and corresponding services/endpoints). As soon as any change in configuration is detected, it registers all exported service endpoints to corresponding (same namespace/service names) AWS Cloud Map structures (namespace, service, instances).
* `pkg/controllers/cloudmap_controller` is periodically polling for changes in corresponding AWS Cloud Map namespaces (based on namespace "sameness" - a K8s namespace with the same name as a Cloud Map namespace). When new service or endpoints are discovered they are automatically created locally as a `ServiceImport`. 

## Getting Started

### Build and Unit Tests

Use command below to run the unit test:
```sh
make test
```

Use command below to build:
```sh
make build
```

Use the command below to perform cleanup:
```sh
make clean
```

### Local Setup

#### Prerequisites

In order to build and run locally:

* Make sure to have `kubectl` [installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/), at least version `1.17` or above.
* Make sure to have `kind` [installed](https://kind.sigs.k8s.io/docs/user/quick-start/#installation).
* Make sure, you have access to AWS Cloud Map. As per exercise below, AWS Cloud Map namespace `example` of the type [HttpNamespace](https://docs.aws.amazon.com/cloud-map/latest/api/API_CreateHttpNamespace.html) will be automatically created.

Note that this walk-through assumes throughout to operate in the `us-west-2` region.

```sh
export AWS_REGION=us-west-2
```

#### Kind Cluster Setup

Spin up a local Kubernetes cluster using `kind`:

```sh
kind create cluster --name my-cluster
# Creating cluster "my-cluster" ...
# ...
```

When completed, set the kubectl context:
```sh
kind export kubeconfig --name my-cluster
# Set kubectl context to "kind-my-cluster"
```

Create `example` namespace in the cluster:
```sh
kubectl create namespace example
# namespace/example created
```

#### Run the controller from outside the cluster

To register the custom CRDs (`ClusterProperties`, `ServiceImport`, `ServiceExport`) in the cluster and create installers:
```sh
make install
# ...
# customresourcedefinition.apiextensions.k8s.io/clusterproperties.about.k8s.io created
# customresourcedefinition.apiextensions.k8s.io/serviceexports.multicluster.x-k8s.io created
# customresourcedefinition.apiextensions.k8s.io/serviceimports.multicluster.x-k8s.io created
```

Register a unique `id.k8s.io` and `clusterset.k8s.io` in your cluster:
```bash
kubectl apply -f samples/example-clusterproperty.yaml
# clusterproperty.about.k8s.io/id.k8s.io created
# clusterproperty.about.k8s.io/clusterset.k8s.io created
```
> ⚠ **Note:** If you are creating multiple clusters, ensure you create unique `id.k8s.io` identifiers for each cluster.


To run the controller, run the following command. The controller runs in an infinite loop so open another terminal to create CRDs. (Ctrl+C to exit)
```sh
make run 
```

Apply deployment, service and serviceexport configs:
```sh
kubectl apply -f samples/example-deployment.yaml
# deployment.apps/nginx-deployment created
kubectl apply -f samples/example-service.yaml
# service/my-service created
kubectl apply -f samples/example-serviceexport.yaml
# serviceexport.multicluster.x-k8s.io/my-service created
```

Check running controller if it correctly detects newly created resources:
```
controllers.ServiceExport	updating Cloud Map service	{"serviceexport": "example/my-service", "namespace": "example", "name": "my-service"}
cloudmap	                fetching a service	{"namespaceName": "example", "serviceName": "my-service"}
cloudmap	                creating a new service	{"namespace": "example", "name": "my-service"}
```

Use the command below to remove the CRDs from the cluster:
```sh
make uninstall
```

#### Build and deploy controller into the cluster

Build local `controller` docker image:
```sh
make docker-build IMG=controller:local
# ...
# docker build --no-cache -t controller:local .
# ...
# 
```

Load the controller docker image into the kind cluster `my-cluster`:
```sh
kind load docker-image controller:local --name my-cluster
# Image: "controller:local" with ID "sha256:xxx" not yet present on node "my-cluster-control-plane", loading...
```

> ⚠ **The controller still needs credentials to interact to AWS SDK.** We are not supporting this configuration out of box. There are multiple ways to achieve this within the cluster.

Finally, create the controller resources in the cluster:
```sh
make deploy IMG=controller:local AWS_REGION=us-west-2
# customresourcedefinition.apiextensions.k8s.io/clusterproperties.about.k8s.io created
# customresourcedefinition.apiextensions.k8s.io/serviceexports.multicluster.x-k8s.io created
# customresourcedefinition.apiextensions.k8s.io/serviceimports.multicluster.x-k8s.io created
# ...
# deployment.apps/cloud-map-mcs-controller-manager created
```

Stream the controller logs:
```shell
kubectl logs -f -l control-plane=controller-manager -c manager -n cloud-map-mcs-system
```

To remove the controller from your cluster, run:
```sh
make undeploy
```

Use the command below to delete the cluster `my-cluster`:
```sh
kind delete cluster --name my-cluster
```

### Local integration testing

The end-to-end integration test suite can be run locally to validate controller core functionality. This will provision a local Kind cluster and build and run the AWS Cloud Map MCS Controller for K8s. The test will verify service endpoints sync with AWS Cloud Map. If successful, the suite will then de-provision the local test cluster and delete AWS Cloud Map namespace `aws-cloud-map-mcs-e2e` along with test service and service instance resources:
```sh
make kind-integration-suite
```

If integration test suite fails for some reason, you can perform a cleanup:
```sh
make kind-integration-cleanup
```

## Build and push docker image

You must first push a Docker image containing the changes to a Docker repository like ECR, Github packages, or DockerHub. The repo is configured to use Github Actions to automatically publish the docker image upon push to `main` branch. The image URI will be `ghcr.io/[Your forked repo name here]` You can enable this for forked repos by enabling Github actions on your forked repo in the "Actions" tab of forked repo.

If you are deploying to cluster using kustomize templates from the `config` directory, you will need to override the image URI away from `ghcr.io/aws/aws-cloud-map-mcs-controller-for-k8s` in order to use your own docker images.

To push the docker image into personal repo:
```sh
make docker-build docker-push IMG=[Your personal repo]
```

## Reporting Bugs/Feature Requests

We welcome you to use the GitHub issue tracker to report bugs or suggest features.

When filing an issue, please check [existing open](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues), or [recently closed](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/issues?utf8=%E2%9C%93&q=is%3Aissue%20is%3Aclosed%20), issues to make sure somebody else hasn't already
reported the issue. Please try to include as much information as you can. Details like these are incredibly useful:

* A reproducible test case or series of steps
* The version of our code being used
* Any modifications you've made relevant to the bug
* Anything unusual about your environment or deployment

## Contributing via Pull Requests

Contributions via pull requests are much appreciated. Before sending us a pull request, please ensure that:

1. You are working against the latest source on the *main* branch.
2. You have checked [existing open](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pulls), and [recently closed](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pulls?q=is%3Apr+is%3Aclosed), pull requests to make sure someone else hasn't addressed the problem already.
3. You have opened an issue to discuss any significant work - we would hate for your time to be wasted.

To send us a pull request, please:

1. Fork the repository.
2. Modify the source; please focus on the specific change you are contributing. If you also reformat all the code, it will be hard for us to focus on your change.
3. Ensure local tests pass.
4. Commit to your fork using clear commit messages.
5. Send us a pull request, answering any default questions in the pull request interface.
6. Pay attention to any automated CI failures reported in the pull request, and stay involved in the conversation.

GitHub provides additional document on [forking a repository](https://help.github.com/articles/fork-a-repo/) and
[creating a pull request](https://help.github.com/articles/creating-a-pull-request/).

## Finding contributions to work on

Looking at the existing issues is a great way to find something to contribute on. As our projects, by default, use the default GitHub issue labels (enhancement/bug/duplicate/help wanted/invalid/question/wontfix), looking at any ["help wanted"](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/labels/help%20wanted) issues is a great place to start.

## Code of Conduct

This project has adopted the [Amazon Open Source Code of Conduct](https://aws.github.io/code-of-conduct).
For more information see the [Code of Conduct FAQ](https://aws.github.io/code-of-conduct-faq) or contact
[opensource-codeofconduct@amazon.com](mailto:opensource-codeofconduct@amazon.com) with any additional questions or comments.

## Security issue notifications

If you discover a potential security issue in this project we ask that you notify AWS/Amazon Security via our [vulnerability reporting page](http://aws.amazon.com/security/vulnerability-reporting/) or [email AWS security directly](mailto:aws-security@amazon.com). Please do **not** create a public github issue.

## Licensing

See the [LICENSE](https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/blob/master/LICENSE) file for our project's licensing. We will ask you to confirm the licensing of your contribution.
