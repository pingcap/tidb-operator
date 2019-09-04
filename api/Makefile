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
TEST_COVER_PACKAGES:=go list ./pkg/... | grep -vE "pkg/client" | grep -vE "pkg/tkctl" | grep -vE "pkg/apis" | sed 's|github.com/pingcap/tidb-operator/|./|' | tr '\n' ','

apiserver-build:
	$(GO) -a -o artifacts/image/tidb-apiserver

apiserver-docker: apiserver-build
	docker build -t "${DOCKER_REGISTRY}/pingcap/tidb-apiserver:latest" ./artifacts/image

apiserver-push: apiserver-docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-apiserver:latest"

.PHONY: apiserver-build
