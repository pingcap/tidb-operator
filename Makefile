GOVER_MAJOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\1/")
GOVER_MINOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\2/")
GO111 := $(shell [ $(GOVER_MAJOR) -gt 1 ] || [ $(GOVER_MAJOR) -eq 1 ] && [ $(GOVER_MINOR) -ge 11 ]; echo $$?)
ifeq ($(GO111), 1)
$(error Please upgrade your Go compiler to 1.11 or higher version)
endif

GOENV  := GO15VENDOREXPERIMENT="1" GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO     := $(GOENV) go build
GOTEST := CGO_ENABLED=0 GO111MODULE=on go test -v -cover

LDFLAGS = $(shell ./hack/version.sh)

DOCKER_REGISTRY := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY),localhost:5000)

PACKAGE_LIST := go list ./... | grep -vE "pkg/client" | grep -vE "zz_generated"
PACKAGES := $$($(PACKAGE_LIST))
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/pingcap/tidb-operator/||'
FILES := $$(find $$($(PACKAGE_DIRECTORIES)) -name "*.go")
FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

default: build

docker-push: docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest"

docker: build
	docker build --tag "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest" images/tidb-operator

build: controller-manager scheduler discovery

controller-manager:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-controller-manager cmd/controller-manager/main.go

scheduler:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-scheduler cmd/scheduler/main.go

discovery:
	$(GO) -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-discovery cmd/discovery/main.go

e2e-docker-push: e2e-docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator-e2e:latest"

e2e-docker: e2e-build
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-operator-e2e:latest" tests/images/e2e

e2e-build:
	$(GOENV) ginkgo build tests/e2e

stability-test-build:
	$(GO) -ldflags '$(LDFLAGS)' -o tests/images/stability-test/bin/stability-test tests/cmd/stability/main.go

stability-test-docker: stability-test-build
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-operator-stability-test:latest" tests/images/stability-test

stability-test-push: stability-test-docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator-stability-test:latest"

test:
	@echo "Run unit tests"
	@$(GOTEST) ./pkg/... -coverprofile=coverage.txt -covermode=atomic && echo "\nUnit tests run successfully!"

check-all: lint check-static check-shadow check-gosec megacheck errcheck

check-setup:
	@which retool >/dev/null 2>&1 || go get github.com/twitchtv/retool
	@GO111MODULE=off retool sync
	# ginkgo and govet doesn't work with retool for Go 1.11
	# so install separately
	@GO111MODULE=on CGO_ENABLED=0 go get github.com/dnephin/govet@4a96d43e39d340b63daa8bc5576985aa599885f6
	@GO111MODULE=on CGO_ENABLED=0 go get github.com/onsi/ginkgo@v1.6.0

check: check-setup lint check-static

check-static:
	@ # Not running vet and fmt through metalinter becauase it ends up looking at vendor
	@echo "gofmt checking"
	gofmt -s -l -w $(FILES) 2>&1| $(FAIL_ON_STDOUT)
	@echo "govet check"
	govet -all $$($(PACKAGE_DIRECTORIES)) 2>&1
	@echo "mispell and ineffassign checking"
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all \
	  --enable misspell \
	  --enable ineffassign \
	  $$($(PACKAGE_DIRECTORIES))

# TODO: megacheck is too slow currently
megacheck:
	@echo "gometalinter megacheck"
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all --deadline 120s \
	  --enable megacheck \
	  $$($(PACKAGE_DIRECTORIES))

# TODO: errcheck is too slow currently
errcheck:
	@echo "gometalinter errcheck"
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all --deadline 120s \
	  --enable errcheck \
	  $$($(PACKAGE_DIRECTORIES))

# TODO: shadow check fails at the moment
check-shadow:
	@echo "govet shadow checking"
	govet -shadow $$($(PACKAGE_DIRECTORIES))

lint:
	@echo "linting"
	CGO_ENABLED=0 retool do revive -formatter friendly -config revive.toml $$($(PACKAGES))

check-gosec:
	@echo "security checking"
	CGO_ENABLED=0 retool do gosec $$($(PACKAGE_DIRECTORIES))

.PHONY: check check-setup check-all build e2e-build
