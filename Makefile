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
GOENV  := CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH)
GO     := $(GOENV) go
GO_BUILD := $(GO) build -trimpath
GO_SUBMODULES = github.com/pingcap/tidb-operator/pkg/apis github.com/pingcap/tidb-operator/pkg/client
GO_SUBMODULE_DIRS = pkg/apis pkg/client

DOCKER_REGISTRY ?= localhost:5000
DOCKER_REPO ?= ${DOCKER_REGISTRY}/pingcap
IMAGE_TAG ?= latest
TEST_COVER_PACKAGES := go list ./cmd/... ./pkg/... $(foreach mod, $(GO_SUBMODULES), $(mod)/...) | grep -vE "pkg/client" | grep -vE "pkg/tkctl" | grep -vE "pkg/apis/pingcap" | sed 's|github.com/pingcap/tidb-operator/|./|' | tr '\n' ','

# NOTE: coverage report generated for E2E tests (with `-c`) may not stable, see
# https://github.com/golang/go/issues/23883#issuecomment-381766556
GO_TEST := $(GO) test -cover -covermode=atomic -coverpkg=$$($(TEST_COVER_PACKAGES))

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
	[ -d tests/images/e2e/tidb-cluster ] && rm -r tests/images/e2e/tidb-cluster || true
	[ -d tests/images/e2e/tidb-backup ] && rm -r tests/images/e2e/tidb-backup || true
	[ -d tests/images/e2e/manifests ] && rm -r tests/images/e2e/manifests || true
	cp -r charts/tidb-operator tests/images/e2e
	cp -r charts/tidb-cluster tests/images/e2e
	cp -r charts/tidb-backup tests/images/e2e
	cp -r charts/tidb-drainer tests/images/e2e
	cp -r manifests tests/images/e2e
	docker build -t "${DOCKER_REPO}/tidb-operator-e2e:${IMAGE_TAG}" tests/images/e2e

e2e-build: ## Build binaries for test
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/ginkgo github.com/onsi/ginkgo/ginkgo
	$(GO_TEST) -c -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/e2e.test ./tests/e2e
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/webhook ./tests/cmd/webhook
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/blockwriter ./tests/cmd/blockwriter
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/mock-prometheus ./tests/cmd/mock-monitor


debug-docker-push: debug-build-docker
	docker push "${DOCKER_REPO}/debug-launcher:latest"
	docker push "${DOCKER_REPO}/tidb-control:latest"
	docker push "${DOCKER_REPO}/tidb-debug:latest"

debug-build-docker: debug-build
	docker build -t "${DOCKER_REPO}/debug-launcher:latest" misc/images/debug-launcher
	docker build -t "${DOCKER_REPO}/tidb-control:latest" misc/images/tidb-control
	docker build -t "${DOCKER_REPO}/tidb-debug:latest" misc/images/tidb-debug

debug-build: ## Build binaries for debug
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o misc/images/debug-launcher/bin/debug-launcher misc/cmd/debug-launcher/main.go

kubekins-e2e-docker:
	docker build -t "${DOCKER_REPO}/kubekins-e2e:latest" tests/images/kubekins-e2e

e2e-examples:
	./hack/e2e-examples.sh

gocovmerge:
	GOBIN=$(shell pwd)/bin/ $(GO) install github.com/zhouqiang-cl/gocovmerge@latest

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
test: TEST_PACKAGES = ./cmd/backup-manager/app ./pkg
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

cli: ## Build tkctl
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tkctl cmd/tkctl/main.go


##@ Release

docker-release:
	docker buildx build --platform linux/amd64,linux/arm64 --push -t "$(DOCKER_REPO)/tidb-operator:$(IMAGE_TAG)" --build-arg "GOPROXY=$(shell go env GOPROXY)" --build-arg "LDFLAGS=$(LDFLAGS)" -f images/tidb-operator/Dockerfile.new .

.PHONY: check check-setup build e2e-build debug-build cli e2e gocovmerge test docker e2e-docker debug-build-docker

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

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
