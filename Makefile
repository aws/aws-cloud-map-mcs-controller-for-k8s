GIT_COMMIT:=$(shell git describe --dirty --always)
GIT_TAG:=$(shell git describe --dirty --always --tags)
PKG:=github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/version

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# AWS Region
AWS_REGION ?= us-east-1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

mod:
	go mod download

tidy:
	go mod tidy

.PHONY: lint
lint: golangci-lint ## Run linter
	$(GOLANGCI_LINT) run

.PHONY: goimports
goimports: goimports-bin ## run goimports updating files in place
	$(GOIMPORTS) -w .

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
KUBEBUILDER_ASSETS?="$(shell $(ENVTEST) use -i $(ENVTEST_KUBERNETES_VERSION) --bin-dir=$(ENVTEST_ASSETS_DIR) -p path)"

.PHONY: test
test: manifests generate generate-mocks fmt vet test-setup goimports lint ## Run tests
	KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS) go test ./... -coverprofile cover.out -covermode=atomic

test-setup: setup-envtest ## Ensure test environment has been downloaded
ifneq ($(shell test -d $(ENVTEST_ASSETS_DIR); echo $$?), 0)
	@echo Setting up K8s test environment...
	mkdir -p ${ENVTEST_ASSETS_DIR}
	$(ENVTEST) use 1.24.x --bin-dir $(ENVTEST_ASSETS_DIR)
endif

kind-integration-suite: ## Provision and run integration tests with cleanup
	make kind-integration-setup && \
	make kind-integration-run && \
	make kind-integration-cleanup

kind-integration-setup: build kind ## Setup the integration test using kind clusters
	@./integration/kind-test/scripts/setup-kind.sh

kind-integration-run: ## Run the integration test controller
	@./integration/kind-test/scripts/run-helper.sh

kind-integration-cleanup: kind  ## Cleanup integration test resources in Cloud Map and local kind cluster
	@./integration/kind-test/scripts/cleanup-kind.sh

eks-integration-suite: ## Provision and run EKS integration tests with cleanup
	make eks-integration-setup && \
	make eks-integration-run && \
	make eks-integration-cleanup

eks-integration-setup: build ## Setup the integration test using EKS clusters
	@./integration/eks-test/scripts/eks-setup.sh

eks-integration-run: ## Run the integration test controller
	@./integration/eks-test/scripts/eks-run-tests.sh

eks-integration-cleanup:  ## Cleanup integration test resources in Cloud Map and EKS cluster
	@./integration/eks-test/scripts/eks-cleanup.sh

eks-test:
	@./integration/eks-test/scripts/eks-test.sh

##@ Build

.DEFAULT: build
build: test ## Build manager binary.
	go build -ldflags="-s -w -X ${PKG}.GitVersion=${GIT_TAG} -X ${PKG}.GitCommit=${GIT_COMMIT}" -o bin/manager main.go

run: test ## Run a controller from your host.
	go run -ldflags="-s -w -X ${PKG}.GitVersion=${GIT_TAG} -X ${PKG}.GitCommit=${GIT_COMMIT}" ./main.go --zap-devel=true --zap-time-encoding=rfc3339 $(ARGS)

docker-build: test ## Build docker image with the manager.
	docker build --no-cache -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

.PHONY: clean
clean:
	@echo Cleaning...
	go clean
	if test -d $(ENVTEST_ASSETS_DIR) ; then chmod -R +w $(ENVTEST_ASSETS_DIR) ; fi
	rm -rf $(MOCKS_DESTINATION)/ bin/ $(ENVTEST_ASSETS_DIR)/ cover.out

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	AWS_REGION=${AWS_REGION} $(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

MOCKS_DESTINATION=mocks
generate-mocks: mockgen
ifneq ($(shell test -d $(MOCKS_DESTINATION); echo $$?), 0)
	@echo Generating mocks...
	$(MOCKGEN) --source pkg/cloudmap/client.go --destination $(MOCKS_DESTINATION)/pkg/cloudmap/client_mock.go --package cloudmap_mock
	$(MOCKGEN) --source pkg/cloudmap/cache.go --destination $(MOCKS_DESTINATION)/pkg/cloudmap/cache_mock.go --package cloudmap_mock
	$(MOCKGEN) --source pkg/cloudmap/operation_poller.go --destination $(MOCKS_DESTINATION)/pkg/cloudmap/operation_poller_mock.go --package cloudmap_mock
	$(MOCKGEN) --source pkg/cloudmap/api.go --destination $(MOCKS_DESTINATION)/pkg/cloudmap/api_mock.go --package cloudmap_mock
	$(MOCKGEN) --source pkg/cloudmap/aws_facade.go --destination $(MOCKS_DESTINATION)/pkg/cloudmap/aws_facade_mock.go --package cloudmap_mock
	$(MOCKGEN) --source integration/janitor/api.go --destination $(MOCKS_DESTINATION)/integration/janitor/api_mock.go --package janitor_mock
	$(MOCKGEN) --source integration/janitor/aws_facade.go --destination $(MOCKS_DESTINATION)/integration/janitor/aws_facade_mock.go --package janitor_mock
endif

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.2)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.5)

ENVTEST = $(shell pwd)/bin/setup-envtest
setup-envtest: ## Download setup-envtest
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

MOCKGEN = $(shell pwd)/bin/mockgen
mockgen: ## Download mockgen
	$(call go-get-tool,$(MOCKGEN),github.com/golang/mock/mockgen@v1.6.0)

GOLANGCI_LINT=$(shell pwd)/bin/golangci-lint
golangci-lint: ## Download golangci-lint
ifneq ($(shell test -f $(GOLANGCI_LINT); echo $$?), 0)
	@echo Getting golangci-lint...
	@mkdir -p $(shell pwd)/bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell pwd)/bin v1.46.2
endif

GOIMPORTS = $(shell pwd)/bin/goimports
goimports-bin: ## Download mockgen
	$(call go-get-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports@v0.1.12)

KIND = $(shell pwd)/bin/kind
kind: ## Download kind
	$(call go-get-tool,$(KIND),sigs.k8s.io/kind@v0.14.0)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
