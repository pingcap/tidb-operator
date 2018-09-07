GOENV  := GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO     := $(GOENV) go
GOTEST := go test -v -cover

LDFLAGS += -X "github.com/pingcap/tidb-operator/version.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tidb-operator/version.GitSHA=$(shell git rev-parse HEAD)"

DOCKER_REGISTRY := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY),localhost:5000)

PACKAGE_LIST := go list ./... | grep -vE "vendor" | grep -vE "pkg/client" | grep -vE "zz_generated"
PACKAGES := $$($(PACKAGE_LIST))
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/pingcap/tidb-operator/||'
FILES := $$(find $$($(PACKAGE_DIRECTORIES)) -name "*.go")
FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

default: build

docker-push: docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest"

docker: build
	docker build --tag "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest" images/tidb-operator

build: controller-manager

controller-manager:
	$(GO) build -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-controller-manager cmd/controller-manager/main.go

e2e-docker-push: e2e-docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator-e2e:latest"

e2e-docker: e2e-build
	mkdir -p images/tidb-operator-e2e/bin
	mv tests/e2e/e2e.test images/tidb-operator-e2e/bin/
	cp -r charts/tidb-operator images/tidb-operator-e2e/
	cp -r charts/tidb-cluster images/tidb-operator-e2e/
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-operator-e2e:latest" images/tidb-operator-e2e

e2e-build:
	$(GOENV) ginkgo build tests/e2e

test:
	@echo "Run unit tests"
	@$(GOTEST) ./pkg/... && echo "\nUnit tests run successfully!"

check-all: lint check-static check-shadow check-gosec megacheck errcheck

check-setup:
	@which retool >/dev/null 2>&1 || go get github.com/twitchtv/retool
	@retool sync

check: check-setup lint check-static

check-static:
	@ # Not running vet and fmt through metalinter becauase it ends up looking at vendor
	@echo "gofmt checking"
	gofmt -s -l -w $(FILES) 2>&1| $(FAIL_ON_STDOUT)
	@echo "govet checking"
	retool do govet -all $$($(PACKAGE_DIRECTORIES)) 2>&1
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
	retool do govet -shadow $$($(PACKAGE_DIRECTORIES))

lint:
	@echo "linting"
	CGO_ENABLED=0 retool do revive -formatter friendly -config revive.toml $$($(PACKAGES))

check-gosec:
	@echo "security checking"
	CGO_ENABLED=0 retool do gosec $$($(PACKAGE_DIRECTORIES))

.PHONY: check check-all build e2e-build
