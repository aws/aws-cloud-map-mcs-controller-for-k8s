# Contributing Guidelines

Thank you for your interest in contributing to our project. Whether it's a bug report, new feature, correction, or additional
documentation, we greatly value feedback and contributions from our community.

Please read through this document before submitting any issues or pull requests to ensure we have all the necessary
information to effectively respond to your bug report or contribution.

## Getting Started

### Build and Run locally

#### Prerequisites

In order to build and run locally:

* Make sure to have `kubectl` [installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/), at least version `1.15` or above.
* Make sure to have `kind` [installed](https://kind.sigs.k8s.io/docs/user/quick-start/#installation).
* Make sure, you have created a [HttpNamespace](https://docs.aws.amazon.com/cloud-map/latest/api/API_CreateHttpNamespace.html) in AWS Cloud Map. The examples below assumes the namespace name to be `demo`

Note that this walk-through assumes throughout to operate in the `us-west-2` region.

```sh
export AWS_REGION=us-west-2
```

#### Cluster provisioning

Spin up a local Kubernetes cluster using `kind`
```sh
kind create cluster --name my-cluster
# Creating cluster "my-cluster" ...
# ...
```

When completed, set the kubectl context
```sh
kind export kubeconfig --name my-cluster
# Set kubectl context to "kind-my-cluster"
```

To register the custom CRDs (`ServiceImport`, `ServiceExport`) in the cluster and create installers
```sh
make install
# ...
# customresourcedefinition.apiextensions.k8s.io/serviceexports.multicluster.x-k8s.io created
# customresourcedefinition.apiextensions.k8s.io/serviceimports.multicluster.x-k8s.io created
```

To run the controller, run the following command. The controller runs in an infinite loop so open another terminal to create CRDs.
```sh
make run 
```

Create `demo` namespace
```sh
kubectl create namespace demo
# namespace/demo created
```

Apply deployment, service and export configs
```sh
kubectl apply -f samples/demo-deployment.yaml
# deployment.apps/nginx-deployment created
kubectl apply -f samples/demo-service.yaml
# service/demo-service created
kubectl apply -f samples/demo-export.yaml
# serviceexport.multicluster.x-k8s.io/demo-service created
```

Check running controller if it correctly detects newly created resources
```
controllers.ServiceExport	updating Cloud Map service	{"serviceexport": "demo/demo-service", "namespace": "demo", "name": "demo-service"}
cloudmap	                fetching a service	{"namespaceName": "demo", "serviceName": "demo-service"}
cloudmap	                creating a new service	{"namespace": "demo", "name": "demo-service"}
```

#### Run unit tests

Use command below to run the unit test
```sh
make test
```

#### Cleanup

Use the command below to clean all the generated files
```sh
make clean
```

Use the command below to delete the cluster `my-cluster`
```sh
kind delete cluster --name my-cluster
```

### Deploying to a cluster

You must first push a Docker image containing the changes to a Docker repository like ECR.

### Build and push docker image to ECR

```sh
make docker-build docker-push IMG=<YOUR ACCOUNT ID>.dkr.ecr.<ECR REGION>.amazonaws.com/<ECR REPOSITORY>
```

#### Deployment

You must specify AWS access credentials for the operator. You can do so via environment variables, or by creating them.

Any one of below three options will work:
```sh
# With an IAM user.
make deploy

# Use an existing access key
OPERATOR_AWS_ACCESS_KEY_ID=xxx OPERATOR_AWS_SECRET_KEY=yyy make deploy

# Use an AWS profile
OPERATOR_AWS_PROFILE=default make deploy
```

#### Uninstallation

To remove the operator from your cluster, run
```sh
make undeploy
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
