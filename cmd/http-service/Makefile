LDFLAGS += -X "github.com/pingcap/tidb-operator/http-service/version.ReleaseVersion=$(shell git describe --tags --dirty="-dev")"
LDFLAGS += -X "github.com/pingcap/tidb-operator/http-service/version.BuildTS=$(shell date -u '+%Y-%m-%d %H:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tidb-operator/http-service/version.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "github.com/pingcap/tidb-operator/http-service/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
LDFLAGS += -X "github.com/pingcap/tidb-operator/http-service/version.GoVersion=$(shell go version)"

GO       := $(GOENV) go
GOARCH   ?= $(shell go env GOARCH)
GOBUILD  := CGO_ENABLED=0 $(GO) build

DOCKER_REGISTRY ?= localhost:5000
DOCKER_REPO     ?= ${DOCKER_REGISTRY}/pingcap
IMAGE_TAG       ?= latest

build:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/$(GOARCH)/http-service .

docker: build
	docker build --tag "${DOCKER_REPO}/tidb-operator-http-service:${IMAGE_TAG}" --build-arg=TARGETARCH=$(GOARCH) .

buf:
	GOBIN=$(shell pwd)/bin/ $(GO) install github.com/bufbuild/buf/cmd/buf@v1.26.1

buf-format: buf
	$(shell pwd)/bin/buf format ./idl/api/service.proto -w

buf-lint: buf buf-format
	$(shell pwd)/bin/buf lint ./idl

buf-generate: buf buf-format
	$(shell pwd)/bin/buf generate ./idl
