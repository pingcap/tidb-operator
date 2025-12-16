LDFLAGS ?= $(shell ./hack/version.sh)
LDFLAGS_PORTS ?= $(shell ./hack/custom-port.sh)
ifeq ($(LDFLAGS_PORTS),)
else
LDFLAGS := $(LDFLAGS) $(LDFLAGS_PORTS)
endif

GOVER_MAJOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\1/")
GOVER_MINOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\2/")
GO113 := $(shell [ $(GOVER_MAJOR) -gt 1 ] || [ $(GOVER_MAJOR) -eq 1 ] && [ $(GOVER_MINOR) -ge 13 ]; echo $$?)
ifeq ($(GO113), 1)
$(error Please upgrade your Go compiler to 1.13 or  higher version)
endif

# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on
GOOS ?= linux
GOARCH ?= $(shell go env GOARCH)
GOENV  := GOOS=$(GOOS) GOARCH=$(GOARCH)
GO     := $(GOENV) go
ifeq ("${ENABLE_FIPS}", "1")
GO_BUILD := GOEXPERIMENT=boringcrypto CGO_ENABLED=1 $(GO) build -trimpath -tags boringcrypto
else
GO_BUILD := CGO_ENABLED=0 $(GO) build -trimpath
endif
GO_SUBMODULES = github.com/pingcap/tidb-operator/pkg/apis github.com/pingcap/tidb-operator/pkg/client
GO_SUBMODULE_DIRS = pkg/apis pkg/client

DOCKER_REGISTRY ?= localhost:5000
DOCKER_REPO ?= ${DOCKER_REGISTRY}/pingcap
IMAGE_TAG ?= latest
TEST_COVER_PACKAGES := go list ./cmd/... ./pkg/... $(foreach mod, $(GO_SUBMODULES), $(mod)/...) | grep -vE "pkg/client" | grep -vE "pkg/apis/pingcap" | sed 's|github.com/pingcap/tidb-operator/|./|' | tr '\n' ','

# NOTE: coverage report generated for E2E tests (with `-c`) may not stable, see
# https://github.com/golang/go/issues/23883#issuecomment-381766556
GO_TEST := CGO_ENABLED=0 $(GO) test -cover -covermode=atomic -coverpkg=$$($(TEST_COVER_PACKAGES))

default: build

ifeq ($(NO_BUILD),y)
operator-docker:
	@echo "NO_BUILD=y, skip build for $@"
else
operator-docker: build
endif
ifeq ($(E2E),y)
	docker build --tag "${DOCKER_REPO}/tidb-operator:${IMAGE_TAG}" -f images/tidb-operator/Dockerfile.e2e images/tidb-operator
else
	docker build --tag "${DOCKER_REPO}/tidb-operator:${IMAGE_TAG}" --build-arg=TARGETARCH=$(GOARCH) images/tidb-operator
endif

build: controller-manager scheduler discovery admission-webhook backup-manager br-federation-manager

##@ Build

controller-manager: ## Build tidb-controller-manager binary
ifeq ($(E2E),y)
	$(GO_TEST) -ldflags '$(LDFLAGS)' -c -o images/tidb-operator/bin/tidb-controller-manager ./cmd/controller-manager
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/$(GOARCH)/tidb-controller-manager cmd/controller-manager/main.go
endif

scheduler: ## Build tidb-scheduler binary
ifeq ($(E2E),y)
	$(GO_TEST) -ldflags '$(LDFLAGS)' -c -o images/tidb-operator/bin/tidb-scheduler ./cmd/scheduler
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/$(GOARCH)/tidb-scheduler cmd/scheduler/main.go
endif

discovery: ## Build tidb-discovery binary
ifeq ($(E2E),y)
	$(GO_TEST) -ldflags '$(LDFLAGS)' -c -o images/tidb-operator/bin/tidb-discovery ./cmd/discovery
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/$(GOARCH)/tidb-discovery cmd/discovery/main.go
endif

admission-webhook: ## Build tidb-admission-webhook binary
ifeq ($(E2E),y)
	$(GO_TEST) -ldflags '$(LDFLAGS)' -c -o images/tidb-operator/bin/tidb-admission-webhook ./cmd/admission-webhook
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/$(GOARCH)/tidb-admission-webhook cmd/admission-webhook/main.go
endif

backup-manager: ## Build tidb-backup-manager binary
ifeq ($(E2E),y)
	$(GO_TEST) -ldflags '$(LDFLAGS)' -c -o images/tidb-backup-manager/bin/tidb-backup-manager ./cmd/backup-manager
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-backup-manager/bin/$(GOARCH)/tidb-backup-manager cmd/backup-manager/main.go
endif

br-federation-manager: ## Build br-federation-manager binary
ifeq ($(E2E),y)
	$(GO_TEST) -ldflags '$(LDFLAGS)' -c -o images/br-federation-manager/bin/br-federation-manager ./cmd/br-federation-manager
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/br-federation-manager/bin/$(GOARCH)/br-federation-manager ./cmd/br-federation-manager
endif

ebs-warmup:
ifeq ($(E2E),y)
	$(GO_TEST) -ldflags '$(LDFLAGS)' -c -o images/ebs-warmup/bin/warmup ./cmd/ebs-warmup
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/ebs-warmup/bin/$(GOARCH)/warmup ./cmd/ebs-warmup
endif

##@ Build Docker images
docker: operator-docker backup-docker br-federation-docker

docker-push: docker ## Push Docker images to registry
	docker push "${DOCKER_REPO}/tidb-operator:${IMAGE_TAG}"
	docker push "${DOCKER_REPO}/tidb-backup-manager:${IMAGE_TAG}"
	docker push "${DOCKER_REPO}/br-federation-manager:${IMAGE_TAG}"

ifeq ($(NO_BUILD),y)
backup-docker:
	@echo "NO_BUILD=y, skip build for $@"
else
backup-docker: backup-manager ## Build tidb-backup-manager image
endif
ifeq ($(E2E),y)
	docker build --tag "${DOCKER_REPO}/tidb-backup-manager:${IMAGE_TAG}" -f images/tidb-backup-manager/Dockerfile.e2e images/tidb-backup-manager
else
	docker build --tag "${DOCKER_REPO}/tidb-backup-manager:${IMAGE_TAG}" --build-arg=TARGETARCH=$(GOARCH) images/tidb-backup-manager
endif

ifeq ($(NO_BUILD),y)
br-federation-docker:
	@echo "NO_BUILD=y, skip build for $@"
else
br-federation-docker: br-federation-manager ## Build br-federation-manager image
endif
ifeq ($(E2E),y)
	docker build --tag "${DOCKER_REPO}/br-federation-manager:${IMAGE_TAG}" -f images/br-federation-manager/Dockerfile.e2e images/br-federation-manager
else
	docker build --tag "${DOCKER_REPO}/br-federation-manager:${IMAGE_TAG}" --build-arg=TARGETARCH=$(GOARCH) images/br-federation-manager
endif

ifeq ($(NO_BUILD),y)
ebs-warmup-docker:
	@echo "NO_BUILD=y, skip build for $@"
else
ebs-warmup-docker: ebs-warmup
endif
ifeq ($(E2E),y)
	docker build --tag "${DOCKER_REPO}/ebs-warmup:${IMAGE_TAG}" -f images/ebs-wamrup/Dockerfile.e2e images/ebs-warmup
else
	docker build --tag "${DOCKER_REPO}/ebs-warmup:${IMAGE_TAG}" --build-arg=TARGETARCH=$(GOARCH) images/ebs-warmup
endif

e2e-docker-push: e2e-docker ## Push tidb-operator-e2e image to registry
	docker push "${DOCKER_REPO}/tidb-operator-e2e:${IMAGE_TAG}"

ifeq ($(NO_BUILD),y)
e2e-docker:
	@echo "NO_BUILD=y, skip build for $@"
else
e2e-docker: e2e-build ## Build tidb-operator-e2e image
endif
	[ -d tests/images/e2e/tidb-operator ] && rm -r tests/images/e2e/tidb-operator || true
	[ -d tests/images/e2e/manifests ] && rm -r tests/images/e2e/manifests || true
	cp -r charts/tidb-operator tests/images/e2e
	cp -r charts/tidb-drainer tests/images/e2e
	cp -r manifests tests/images/e2e
	docker build -t "${DOCKER_REPO}/tidb-operator-e2e:${IMAGE_TAG}" --build-arg=TARGETARCH=$(GOARCH) tests/images/e2e

e2e-build: ## Build binaries for test
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/ginkgo github.com/onsi/ginkgo/ginkgo
	$(GO_TEST) -c -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/e2e.test ./tests/e2e
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/webhook ./tests/cmd/webhook
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/blockwriter ./tests/cmd/blockwriter

debug-docker-push: debug-build-docker
	docker push "${DOCKER_REPO}/tidb-control:latest"

debug-build-docker:
	docker build -t "${DOCKER_REPO}/tidb-control:latest" misc/images/tidb-control

kubekins-e2e-docker:
	docker build -t "${DOCKER_REPO}/kubekins-e2e:latest" tests/images/kubekins-e2e

e2e-examples:
	./hack/e2e-examples.sh

gocovmerge:
	mkdir -p bin
	$(GO_BUILD) -o bin/gocovmerge ./pkg/third_party/gocovmerge

fault-trigger:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/fault-trigger/bin/fault-trigger tests/cmd/fault-trigger/*.go

##@ Tests

# ARGS:
#
# GOFLAGS: Extra flags to pass to 'go' when building, e.g.
# 		`-v` for verbose logging.
# 		`-race` for race detector.
# GO_COVER: Whether to run tests with code coverage. Set to 'y' to enable coverage collection.
#
test: TEST_PACKAGES = ./cmd/backup-manager/app ./pkg ./cmd/ebs-warmup/internal/tests
test: ## Run unit tests
	@echo "Run unit tests"
ifeq ($(GO_COVER),y)
	go test -cover \
		$(foreach pkg, $(TEST_PACKAGES), $(pkg)/...) \
		$(foreach mod, $(GO_SUBMODULES), $(mod)/...) \
		-coverpkg=$$($(TEST_COVER_PACKAGES)) -coverprofile=coverage.txt -covermode=atomic
else
	go test \
		$(foreach pkg, $(TEST_PACKAGES), $(pkg)/...) \
		$(foreach mod, $(GO_SUBMODULES), $(mod)/...)
endif
	@echo -e "\nUnit tests run successfully!"

e2e: ## Run e2e tests
	./hack/e2e.sh

##@ Development

ALL_CHECKS = EOF codegen crd boilerplate openapi-spec api-references spelling modules lint
check: $(addprefix check-,$(ALL_CHECKS)) tidy

check-%: ## Check code style, spelling and lint
	./hack/verify-$*.sh

generate: ## Generate code
	./hack/update-all.sh

crd: ## Generate CRD manifests
	./hack/update-crd.sh

api-references: ## Generate API references
	./hack/update-api-references.sh

tidy: ## Run go mod tidy and verify the result
	@echo "go mod tidy"
	go mod tidy && git diff -U --exit-code go.mod go.sum
	cd pkg/apis && go mod tidy && git diff -U --exit-code go.mod go.sum
	cd pkg/client && go mod tidy && git diff -U --exit-code go.mod go.sum


##@ Release

docker-release:
	docker buildx build --platform linux/amd64,linux/arm64 --push -t "$(DOCKER_REPO)/tidb-operator:$(IMAGE_TAG)" --build-arg "GOPROXY=$(shell go env GOPROXY)" --build-arg "LDFLAGS=$(LDFLAGS)" -f images/tidb-operator/Dockerfile.new .

.PHONY: check check-setup build e2e-build e2e gocovmerge test docker e2e-docker debug-build-docker

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

<<<<<<< HEAD
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
=======
.PHONY: runtimegen
runtimegen: bin/runtime-gen
	$(RUNTIME_GEN) \
		--output-dir=$(RUNTIME_PKG_DIR) \
		--go-header-file=$(BOILERPLATE_FILE) \
		github.com/pingcap/tidb-operator/api/v2/core/v1alpha1

.PHONY: release
release: bin/helm crd
	$(ROOT)/hack/release.sh

.PHONY: doc
doc: bin/mdtoc
	find docs -name "*.md" | xargs $(MDTOC) --inplace --max-depth 5

.PHONY: crd
crd: bin/controller-gen build/crd-modifier
	$(CONTROLLER_GEN) crd:generateEmbeddedObjectMeta=true output:crd:artifacts:config=$(ROOT)/manifests/crd paths=$(API_PATH)/...
	$(BIN_DIR)/crd-modifier -dir $(ROOT)/manifests/crd


.PHONY: tidy
tidy: $(addprefix tidy/,$(GO_TOOL_BIN))
	cd $(API_PATH) && go mod tidy
	cd $(VALIDATION_TEST_PATH) && go mod tidy
	go mod tidy

.PHONY: $(addprefix tidy/,$(GO_TOOL_BIN))
$(addprefix tidy/,$(GO_TOOL_BIN)):
	cd $(TOOLS_PATH)/$(patsubst tidy/%,%,$@) && go mod tidy

gengo: GEN_DIR ?= ./...
gengo: bin/mockgen
	BOILERPLATE_FILE=${MOCK_BOILERPLATE_FILE} GOBIN=$(BIN_DIR) GO_MODULE=$(GO_MODULE) go generate $(GEN_DIR)

.PHONY: license
license: bin/license-eye
	$(LICENSE_EYE) -c .github/licenserc.yaml header fix

ALL_GEN = tidy codegen crd runtimegen gengo overlaygen doc
.PHONY: generate
generate: $(ALL_GEN) license

.PHONY: verify/license
verify/license: bin/license-eye
	$(LICENSE_EYE) -c .github/licenserc.yaml header check

.PHONY: verify/feature-gates
verify/feature-gates:
	cd $(ROOT) && go run cmd/verify-feature-gates/main.go

.PHONY: verify
verify: $(addprefix verify/,$(ALL_GEN)) verify/license verify/feature-gates
verify/%:
	$(ROOT)/hack/verify.sh make $*

.PHONY: lint
lint: bin/golangci-lint
	$(GOLANGCI_LINT) run -v ./...

.PHONY: lint-fix
lint-fix: bin/golangci-lint
	$(GOLANGCI_LINT) run -v ./... --fix

.PHONY: unit
unit:
	cd $(VALIDATION_TEST_PATH) && go test -race ./...
	go test -race $$(go list -e ./... | grep -v cmd | grep -v tools | grep -v tests/e2e | grep -v third_party) \
		-cover -coverprofile=coverage.txt -covermode=atomic
	sed -i.bak '/generated/d;/fake.go/d' coverage.txt && rm coverage.txt.bak

.PHONY: check
check: lint unit verify

.PHONY: install-githooks
install-githooks:
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@ln -sf ../../hack/githooks/pre-push .git/hooks/pre-push
	@echo "pre-push hook installed successfully."
	@echo "You can run 'make check' manually to check your code before push."

.PHONY: e2e/prepare
e2e/prepare: bin/kind release
	$(ROOT)/hack/e2e.sh --prepare

# e2e/run: Run e2e tests (excluding packages specified in E2E_EXCLUDED_PACKAGES)
# Default excludes 'upgrade' package which requires special build tags
# Usage: make e2e/run
# To exclude additional packages: E2E_EXCLUDED_PACKAGES="upgrade,some-other-package" make e2e/run
# To run all packages: E2E_EXCLUDED_PACKAGES="" make e2e/run
.PHONY: e2e/run
e2e/run:
	$(ROOT)/hack/e2e.sh run $(GINKGO_OPTS)

.PHONY: e2e/run-upgrade
e2e/run-upgrade:
	$(ROOT)/hack/e2e.sh run-upgrade $(GINKGO_OPTS)

# e2e: Run full e2e test suite including both regular and upgrade tests
.PHONY: e2e
e2e: bin/kind release
	$(ROOT)/hack/e2e.sh --prepare run run-upgrade $(GINKGO_OPTS)

.PHONY: e2e/deploy
e2e/deploy: bin/kubectl release
	$(KUBECTL) $(KUBE_OPT) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator.crds.yaml
	$(KUBECTL) $(KUBE_OPT) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator-e2e.yaml

.PHONY: kube
kube: bin/kind bin/kubectl
	@echo "ensure that the kubernetes env is existing"
	V_KIND=$(KIND) V_KUBECTL=$(KUBECTL) $(ROOT)/hack/kind.sh

.PHONY: reload/operator
reload/operator: bin/kubectl
	$(KUBECTL) $(KUBE_OPT) delete pod `$(KUBECTL) $(KUBE_OPT) get pods | awk '/operator/{ print $$1 }'`

.PHONY: logs/operator
logs/operator: bin/kubectl
	$(KUBECTL) $(KUBE_OPT) logs -f `$(KUBECTL) $(KUBE_OPT) get pods | awk '/operator/{ print $$1 }'`

OVERLAY_GEN = $(BIN_DIR)/overlay-gen
bin/overlay-gen:
	$(ROOT)/hack/build.sh overlay-gen

RUNTIME_GEN = $(BIN_DIR)/runtime-gen
bin/runtime-gen:
	$(ROOT)/hack/build.sh runtime-gen

# Generic target for allowed bin/xxx tools - automatically defines XXX variable (with hyphens converted to underscores)
# e.g. bin/abc-def will define ABC_DEF = $(BIN_DIR)/abc-def
define make_bin_target
$(eval $(shell echo $(1) | tr '[:lower:]-' '[:upper:]_') = $(BIN_DIR)/$(1))
endef

.PHONY: $(addprefix bin/,$(GO_TOOL_BIN))
$(addprefix bin/,$(GO_TOOL_BIN)): bin/%: tidy/%
	$(call make_bin_target,$(patsubst bin/%,%,$@))
	./hack/tools.sh $(patsubst bin/%,%,$@)


.PHONY: charts/build
charts/build:
	$(ROOT)/hack/charts-build.sh
>>>>>>> b2e036e9d (feat: add new github action for tidb-operator-release v1.x (#6606))
