GOENV     := GO15VENDOREXPERIMENT="1" GOOS=linux GOARCH=amd64
GO        := $(GOENV) CGO_ENABLED=0 go

LDFLAGS += -X "github.com/pingcap/tidb-operator/new-operator/version.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tidb-operator/new-operator/version.GitSHA=$(shell git rev-parse HEAD)"

DOCKER_REGISTRY := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY),localhost:5000)

default: build

docker-push: docker
	docker push "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest"

docker: build
	docker build --tag "${DOCKER_REGISTRY}/pingcap/tidb-operator:latest" images/tidb-operator

build: controller-manager

controller-manager:
	$(GO) build -ldflags '$(LDFLAGS)' -o images/tidb-operator/bin/tidb-controller-manager cmd/controller-manager/main.go

test:
	@ CGO_ENABLED=0 go test ./pkg/... -v -cover && echo success
