# Set DEBUGGER=1 to build debug symbols
LDFLAGS = $(if $(DEBUGGER),,-s -w) $(shell ./hack/version.sh)

GOVER_MAJOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\1/")
GOVER_MINOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\2/")
GO113 := $(shell [ $(GOVER_MAJOR) -gt 1 ] || [ $(GOVER_MAJOR) -eq 1 ] && [ $(GOVER_MINOR) -ge 13 ]; echo $$?)
ifeq ($(GO113), 1)
$(error Please upgrade your Go compiler to 1.13 or higher version)
endif

# Enable GO111MODULE=on explicitly, disable it with GO111MODULE=off when necessary.
export GO111MODULE := on
GOOS := $(if $(GOOS),$(GOOS),linux)
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GOENV  := GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH)
GO     := $(GOENV) go
GO_BUILD := $(GO) build -trimpath

DOCKER_REGISTRY ?= localhost:5000
DOCKER_REPO ?= ${DOCKER_REGISTRY}/pingcap
IMAGE_TAG ?= latest
TEST_COVER_PACKAGES:=go list ./cmd/... ./pkg/... | grep -vE "pkg/client" | grep -vE "pkg/tkctl" | grep -vE "pkg/apis" | sed 's|github.com/pingcap/tidb-operator/|./|' | tr '\n' ','

GOTEST := $(GO) test -cover -covermode=atomic -coverpkg=$$($(TEST_COVER_PACKAGES))

default: build

docker-push: docker backup-docker
	docker push "${DOCKER_REPO}/tidb-operator:${IMAGE_TAG}"
	docker push "${DOCKER_REPO}/tidb-backup-manager:${IMAGE_TAG}"

ifeq ($(NO_BUILD),y)
docker:
	@echo "NO_BUILD=y, skip build for $@"
else
docker: build
endif
ifeq ($(E2E),y)
	docker build --tag "${DOCKER_REPO}/tidb-operator:${IMAGE_TAG}" -f images/tidb-operator/Dockerfile.e2e images/tidb-operator
else
	docker build --tag "${DOCKER_REPO}/tidb-operator:${IMAGE_TAG}" images/tidb-operator
endif
	docker build --tag "${DOCKER_REPO}/tidb-backup-manager:${IMAGE_TAG}" images/tidb-backup-manager

build: controller-manager scheduler discovery admission-webhook backup-manager

controller-manager:
ifeq ($(E2E),y)
	$(GOTEST) -c -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-controller-manager ./cmd/controller-manager
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-controller-manager ./cmd/controller-manager
endif

scheduler:
ifeq ($(E2E),y)
	$(GOTEST) -c -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-scheduler ./cmd/scheduler
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-scheduler ./cmd/scheduler
endif

discovery:
ifeq ($(E2E),y)
	$(GOTEST) -c -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-discovery ./cmd/discovery
else
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-discovery ./cmd/discovery
endif

admission-webhook:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-admission-webhook ./cmd/admission-webhook

backup-manager:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o images/tidb-backup-manager/bin/tidb-backup-manager ./cmd/backup-manager

ifeq ($(NO_BUILD),y)
backup-docker:
	@echo "NO_BUILD=y, skip build for $@"
else
backup-docker: backup-manager
endif
	docker build --tag "${DOCKER_REPO}/tidb-backup-manager:${IMAGE_TAG}" images/tidb-backup-manager

e2e-docker-push: e2e-docker
	docker push "${DOCKER_REPO}/tidb-operator-e2e:${IMAGE_TAG}"

ifeq ($(NO_BUILD),y)
e2e-docker:
	@echo "NO_BUILD=y, skip build for $@"
else
e2e-docker: e2e-build
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

e2e-build:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/ginkgo github.com/onsi/ginkgo/ginkgo
	$(GOTEST) -c -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/e2e.test ./tests/e2e
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/webhook ./tests/cmd/webhook
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/blockwriter ./tests/cmd/blockwriter
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/mock-prometheus ./tests/cmd/mock-monitor

e2e:
	./hack/e2e.sh

e2e-examples:
	./hack/e2e-examples.sh

stability-test-build:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/stability-test/bin/blockwriter ./tests/cmd/blockwriter
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/stability-test/bin/stability-test ./tests/cmd/stability

stability-test-docker: stability-test-build
	docker build -t "${DOCKER_REPO}/tidb-operator-stability-test:${IMAGE_TAG}" tests/images/stability-test

stability-test-push: stability-test-docker
	docker push "${DOCKER_REPO}/tidb-operator-stability-test:${IMAGE_TAG}"

fault-trigger:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tests/images/fault-trigger/bin/fault-trigger tests/cmd/fault-trigger/*.go

# ARGS:
#
# GOFLAGS: Extra flags to pass to 'go' when building, e.g.
# 		`-v` for verbose logging.
# 		`-race` for race detector.
# GO_COVER: Whether to run tests with code coverage. Set to 'y' to enable coverage collection.
#
ifeq ($(GO_COVER),y)
test:
	@echo "Run unit tests"
	@go test -cover -covermode=atomic -coverprofile=coverage.txt -coverpkg=$$($(TEST_COVER_PACKAGES)) ./cmd/backup-manager/app/... ./pkg/... && echo -e "\nUnit tests run successfully!"
else
test:
	@echo "Run unit tests"
	@go test ./cmd/backup-manager/app/... ./pkg/... && echo -e "\nUnit tests run successfully!"
endif

ALL_CHECKS = EOF codegen terraform boilerplate openapi-spec crd-groups spelling

check: $(addprefix check-,$(ALL_CHECKS)) lint tidy

check-%:
	./hack/verify-$*.sh

lint:
	./hack/verify-lint.sh

tidy:
	@echo "go mod tidy"
	go mod tidy
	git diff -U --exit-code go.mod go.sum

cli:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o tkctl cmd/tkctl/main.go

debug-docker-push: debug-build-docker
	docker push "${DOCKER_REPO}/debug-launcher:latest"
	docker push "${DOCKER_REPO}/tidb-control:latest"
	docker push "${DOCKER_REPO}/tidb-debug:latest"

debug-build-docker: debug-build
	docker build -t "${DOCKER_REPO}/debug-launcher:latest" misc/images/debug-launcher
	docker build -t "${DOCKER_REPO}/tidb-control:latest" misc/images/tidb-control
	docker build -t "${DOCKER_REPO}/tidb-debug:latest" misc/images/tidb-debug

debug-build:
	$(GO_BUILD) -ldflags '$(LDFLAGS)' -o misc/images/debug-launcher/bin/debug-launcher misc/cmd/debug-launcher/main.go

.PHONY: check check-setup build e2e-build debug-build cli e2e test docker e2e-docker debug-build-docker
