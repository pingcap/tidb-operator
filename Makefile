# Set DEBUGGER=1 to build debug symbols
LDFLAGS = $(if $(DEBUGGER),,-s -w) $(shell ./hack/version.sh)

# SET DOCKER_REGISTRY to change the docker registry
DOCKER_REGISTRY := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY),localhost:5000)

GOVER_MAJOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\1/")
GOVER_MINOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\2/")
GO111 := $(shell [ $(GOVER_MAJOR) -gt 1 ] || [ $(GOVER_MAJOR) -eq 1 ] && [ $(GOVER_MINOR) -ge 11 ]; echo $$?)
ifeq ($(GO111), 1)
$(error Please upgrade your Go compiler to 1.11 or higher version)
endif

GOOS := $(if $(GOOS),$(GOOS),linux)
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GOENV  := GO15VENDOREXPERIMENT="1" GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH)
GO     := $(GOENV) go build
GOTEST := CGO_ENABLED=0 GO111MODULE=on go test -v -cover

PACKAGE_LIST := go list ./... | grep -vE "pkg/client" | grep -vE "zz_generated"
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/pingcap/tidb-operator/||'
FILES := $$(find $$($(PACKAGE_DIRECTORIES)) -name "*.go")
FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

default: build

docker-push: docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest"

docker: build
	docker build --tag "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest" images/tidb-operator

build: controller-manager scheduler discovery admission-controller wait-for-pd

controller-manager:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-controller-manager cmd/controller-manager/main.go

scheduler:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-scheduler cmd/scheduler/main.go

discovery:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-discovery cmd/discovery/main.go

admission-controller:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-admission-controller cmd/admission-controller/main.go

wait-for-pd:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/wait-for-pd cmd/wait-for-pd/main.go

e2e-setup:
	# ginkgo doesn't work with retool for Go 1.11
	@GO111MODULE=on CGO_ENABLED=0 go get github.com/onsi/ginkgo@v1.6.0

e2e-docker-push: e2e-docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator-e2e:latest"

e2e-docker: e2e-build
	[ -d tests/images/e2e/tidb-operator ] && rm -r tests/images/e2e/tidb-operator || true
	[ -d tests/images/e2e/tidb-cluster ] && rm -r tests/images/e2e/tidb-cluster || true
	[ -d tests/images/e2e/tidb-backup ] && rm -r tests/images/e2e/tidb-backup || true
	[ -d tests/images/e2e/manifests ] && rm -r tests/images/e2e/manifests || true
	cp -r charts/tidb-operator tests/images/e2e
	cp -r charts/tidb-cluster tests/images/e2e
	cp -r charts/tidb-backup tests/images/e2e
	cp -r manifests tests/images/e2e
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-operator-e2e:latest" tests/images/e2e

e2e-build: e2e-setup
	$(GO) -ldflags '$(LDFLAGS)' -o tests/images/e2e/bin/e2e tests/cmd/e2e/main.go

stability-test-build:
	$(GO) -ldflags '$(LDFLAGS)' -o tests/images/stability-test/bin/stability-test tests/cmd/stability/*.go

stability-test-docker: stability-test-build
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-operator-stability-test:latest" tests/images/stability-test

stability-test-push: stability-test-docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator-stability-test:latest"

fault-trigger:
	$(GO) -ldflags '$(LDFLAGS)' -o tests/images/fault-trigger/bin/fault-trigger tests/cmd/fault-trigger/*.go

test:
	@echo "Run unit tests"
	@$(GOTEST) ./pkg/... -coverprofile=coverage.txt -covermode=atomic && echo "\nUnit tests run successfully!"

check-all: lint check-static check-shadow check-gosec staticcheck errcheck

check-setup:
	@which retool >/dev/null 2>&1 || go get github.com/twitchtv/retool
	@GO111MODULE=off retool sync

check: check-setup lint tidy check-static

check-static:
	@ # Not running vet and fmt through metalinter becauase it ends up looking at vendor
	@echo "gofmt checking"
	gofmt -s -l -w $(FILES) 2>&1| $(FAIL_ON_STDOUT)
	@echo "go vet check"
	@GO111MODULE=on go vet -all $$($(PACKAGE_LIST)) 2>&1
	@echo "mispell and ineffassign checking"
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all \
	  --enable misspell \
	  --enable ineffassign \
	  $$($(PACKAGE_DIRECTORIES))

# TODO: staticcheck is too slow currently
staticcheck:
	@echo "gometalinter staticcheck"
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all --deadline 120s \
	  --enable staticcheck \
	  $$($(PACKAGE_DIRECTORIES))

# TODO: errcheck is too slow currently
errcheck:
	@echo "gometalinter errcheck"
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all --deadline 120s \
	  --enable errcheck \
	  $$($(PACKAGE_DIRECTORIES))

# TODO: shadow check fails at the moment
check-shadow:
	@echo "go vet shadow checking"
	go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
	@GO111MODULE=on go vet -vettool=$(which shadow) $$($(PACKAGE_LIST))

lint:
	@echo "linting"
	CGO_ENABLED=0 retool do revive -formatter friendly -config revive.toml $$($(PACKAGE_LIST))

tidy:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	git diff --quiet go.mod go.sum

check-gosec:
	@echo "security checking"
	CGO_ENABLED=0 retool do gosec $$($(PACKAGE_DIRECTORIES))

cli:
	$(GO) -ldflags '$(LDFLAGS)' -o tkctl cmd/tkctl/main.go

debug-docker-push: debug-build-docker
	docker push "${DOCKER_REGISTRY}/pingcap/debug-launcher:latest"
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-control:latest"
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-debug:latest"

debug-build-docker: debug-build
	docker build -t "${DOCKER_REGISTRY}/pingcap/debug-launcher:latest" misc/images/debug-launcher
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-control:latest" misc/images/tidb-control
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-debug:latest" misc/images/tidb-debug

debug-build:
	$(GO) -ldflags '$(LDFLAGS)' -o misc/images/debug-launcher/bin/debug-launcher misc/cmd/debug-launcher/main.go

.PHONY: check check-setup check-all build e2e-build debug-build cli
