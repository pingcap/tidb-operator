GOENV     := GO15VENDOREXPERIMENT="1" CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO        := $(GOENV) go

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
	@ CGO_ENABLED=0 go test ./pkg/... -v -cover && echo success
