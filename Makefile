# Copyright 2024 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ROOT = $(CURDIR)
OUTPUT_DIR = $(ROOT)/_output
BIN_DIR = $(OUTPUT_DIR)/bin
API_PATH = $(ROOT)/api
PD_API_PATH = $(ROOT)/pkg/timanager/apis/pd
VALIDATION_TEST_PATH = $(ROOT)/tests/validation
GO_MODULE := github.com/pingcap/tidb-operator
OVERLAY_PKG_DIR = $(ROOT)/pkg/overlay
RUNTIME_PKG_DIR = $(ROOT)/pkg/runtime
BOILERPLATE_FILE = $(ROOT)/hack/boilerplate/boilerplate.go.txt
MOCK_BOILERPLATE_FILE = $(ROOT)/hack/boilerplate/boilerplate.txt

KIND_VERSION ?= v0.24.0

# TODO: use kubectl in _output
KUBECTL = kubectl -n tidb-admin --context kind-tidb-operator

ALL_CMD = tidb-operator prestop-checker testing-workload tidb-backup-manager
.PHONY: build
build: $(addprefix build/,$(ALL_CMD))
build/%:
	$(ROOT)/hack/build.sh $*

.PHONY: image
image: $(addprefix image/,$(ALL_CMD))
image/%:
	$(ROOT)/hack/image.sh $*

.PHONY: push
push: $(addprefix push/,$(ALL_CMD))
push/%:
	$(ROOT)/hack/image.sh $* --push

# Development build targets for faster iteration
.PHONY: dev-image
dev-image: $(addprefix dev-image/,$(ALL_CMD))

# Test target to debug
.PHONY: test-dev
test-dev:
	@echo "Test dev target running"
	$(ROOT)/hack/image.sh tidb-operator --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev

# Force rebuild development images (always execute, ignore file timestamps)
.PHONY: dev-image-tidb-operator dev-image-prestop-checker dev-image-testing-workload dev-image-tidb-backup-manager
dev-image-tidb-operator:
	@echo "Building dev image for tidb-operator..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/tidb-operator" ]; then \
		echo "Binary _output/bin/tidb-operator not found, building..."; \
		$(ROOT)/hack/build.sh tidb-operator; \
	fi
	$(ROOT)/hack/image.sh tidb-operator --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev

dev-image-prestop-checker:
	@echo "Building dev image for prestop-checker..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/prestop-checker" ]; then \
		echo "Binary _output/bin/prestop-checker not found, building..."; \
		$(ROOT)/hack/build.sh prestop-checker; \
	fi
	$(ROOT)/hack/image.sh prestop-checker --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev

# Keep the pattern rule for backwards compatibility but make it work
.PHONY: dev-image/tidb-operator dev-image/prestop-checker dev-image/testing-workload dev-image/tidb-backup-manager
dev-image/tidb-operator: dev-image-tidb-operator
dev-image/prestop-checker: dev-image-prestop-checker
dev-image/testing-workload: dev-image-testing-workload  
dev-image/tidb-backup-manager: dev-image-tidb-backup-manager

dev-image-testing-workload:
	@echo "Building dev image for testing-workload..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/testing-workload" ]; then \
		echo "Binary _output/bin/testing-workload not found, building..."; \
		$(ROOT)/hack/build.sh testing-workload; \
	fi
	$(ROOT)/hack/image.sh testing-workload --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev

dev-image-tidb-backup-manager:
	@echo "Building dev image for tidb-backup-manager..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/tidb-backup-manager" ]; then \
		echo "Binary _output/bin/tidb-backup-manager not found, building..."; \
		$(ROOT)/hack/build.sh tidb-backup-manager; \
	fi
	$(ROOT)/hack/image.sh tidb-backup-manager --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev

.PHONY: dev-push
dev-push: $(addprefix dev-push/,$(ALL_CMD))

# Force rebuild and push development images 
.PHONY: dev-push-tidb-operator dev-push-prestop-checker dev-push-testing-workload dev-push-tidb-backup-manager
dev-push-tidb-operator:
	@echo "Building and pushing dev image for tidb-operator..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/tidb-operator" ]; then \
		echo "Binary _output/bin/tidb-operator not found, building..."; \
		$(ROOT)/hack/build.sh tidb-operator; \
	fi
	$(ROOT)/hack/image.sh tidb-operator --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev --push

dev-push-prestop-checker:
	@echo "Building and pushing dev image for prestop-checker..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/prestop-checker" ]; then \
		echo "Binary _output/bin/prestop-checker not found, building..."; \
		$(ROOT)/hack/build.sh prestop-checker; \
	fi
	$(ROOT)/hack/image.sh prestop-checker --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev --push

dev-push-testing-workload:
	@echo "Building and pushing dev image for testing-workload..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/testing-workload" ]; then \
		echo "Binary _output/bin/testing-workload not found, building..."; \
		$(ROOT)/hack/build.sh testing-workload; \
	fi
	$(ROOT)/hack/image.sh testing-workload --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev --push

dev-push-tidb-backup-manager:
	@echo "Building and pushing dev image for tidb-backup-manager..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "_output/bin/tidb-backup-manager" ]; then \
		echo "Binary _output/bin/tidb-backup-manager not found, building..."; \
		$(ROOT)/hack/build.sh tidb-backup-manager; \
	fi
	$(ROOT)/hack/image.sh tidb-backup-manager --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev --push

# Keep the pattern rule for backwards compatibility but make it work
.PHONY: dev-push/tidb-operator dev-push/prestop-checker dev-push/testing-workload dev-push/tidb-backup-manager
dev-push/tidb-operator: dev-push-tidb-operator
dev-push/prestop-checker: dev-push-prestop-checker
dev-push/testing-workload: dev-push-testing-workload  
dev-push/tidb-backup-manager: dev-push-tidb-backup-manager

# Ensure _output directory is a real directory with binaries, not a symlink
.PHONY: dev-prepare-output
dev-prepare-output:
	@if [ -L "_output" ]; then \
		echo "Converting _output symlink to real directory for Docker build..."; \
		rm _output; \
		mkdir -p _output; \
		cp -r ../tidb-operator/_output/* _output/; \
	elif [ ! -d "_output" ]; then \
		echo "Creating _output directory..."; \
		mkdir -p _output; \
		if [ -d "../tidb-operator/_output" ]; then \
			cp -r ../tidb-operator/_output/* _output/; \
		fi; \
	fi

.PHONY: deploy
deploy: release
	$(KUBECTL) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator.crds.yaml
	$(KUBECTL) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator.yaml

.PHONY: codegen
codegen: bin/deepcopy-gen bin/register-gen bin/overlay-gen
	$(REGISTER_GEN) \
		--output-file=zz_generated.register.go \
		--go-header-file=$(BOILERPLATE_FILE) \
		$(API_PATH)/...

	$(DEEPCOPY_GEN) \
		--output-file=zz_generated.deepcopy.go \
		--go-header-file=$(BOILERPLATE_FILE) \
		$(API_PATH)/...

	$(REGISTER_GEN) \
		--output-file=zz_generated.register.go \
		--go-header-file=$(BOILERPLATE_FILE) \
		$(PD_API_PATH)/...

	$(DEEPCOPY_GEN) \
		--output-file=zz_generated.deepcopy.go \
		--go-header-file=$(BOILERPLATE_FILE) \
		$(PD_API_PATH)/...

.PHONY: overlaygen
overlaygen: bin/overlay-gen
	$(OVERLAY_GEN) \
		--output-dir=$(OVERLAY_PKG_DIR) \
		--go-header-file=$(BOILERPLATE_FILE) \
		k8s.io/api/core/v1

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
tidy:
	cd $(API_PATH) && go mod tidy
	cd $(VALIDATION_TEST_PATH) && go mod tidy
	go mod tidy

gengo: GEN_DIR ?= ./...
gengo: bin/mockgen
	BOILERPLATE_FILE=${MOCK_BOILERPLATE_FILE} GOBIN=$(BIN_DIR) GO_MODULE=$(GO_MODULE) go generate $(GEN_DIR)

.PHONY: license
license: bin/license-eye
	$(LICENSE_EYE) -c .github/licenserc.yaml header fix

ALL_GEN = tidy codegen crd gengo overlaygen runtimegen doc
.PHONY: generate
generate: $(ALL_GEN) license

.PHONY: verify/license
verify/license: bin/license-eye
	$(LICENSE_EYE) -c .github/licenserc.yaml header check

.PHONY: verify
verify: $(addprefix verify/,$(ALL_GEN)) verify/license
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
e2e/deploy: release
	$(KUBECTL) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator.crds.yaml
	$(KUBECTL) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator-e2e.yaml

.PHONY: kube
kube: bin/kind
	@echo "ensure that the kubernetes env is existing"
	$(ROOT)/hack/kind.sh

.PHONY: reload/operator
reload/operator:
	$(KUBECTL) delete pod `$(KUBECTL) get pods | awk '/operator/{ print $$1 }'`

.PHONY: logs/operator
logs/operator:
	$(KUBECTL) logs -f `$(KUBECTL) get pods | awk '/operator/{ print $$1 }'`

CONTROLLER_GEN = $(BIN_DIR)/controller-gen
bin/controller-gen:
	$(ROOT)/hack/download.sh go_install $(CONTROLLER_GEN) sigs.k8s.io/controller-tools/cmd/controller-gen v0.17.2 "--version | awk '{print \$$2}'"

DEEPCOPY_GEN = $(BIN_DIR)/deepcopy-gen
bin/deepcopy-gen:
	$(ROOT)/hack/download.sh go_install $(DEEPCOPY_GEN) k8s.io/code-generator/cmd/deepcopy-gen

REGISTER_GEN = $(BIN_DIR)/register-gen
bin/register-gen:
	$(ROOT)/hack/download.sh go_install $(REGISTER_GEN) k8s.io/code-generator/cmd/register-gen

MOCKGEN = $(BIN_DIR)/mockgen
bin/mockgen:
	$(ROOT)/hack/download.sh go_install $(MOCKGEN) go.uber.org/mock/mockgen v0.5.0 "--version"

OVERLAY_GEN = $(BIN_DIR)/overlay-gen
bin/overlay-gen:
	$(ROOT)/hack/build.sh overlay-gen

RUNTIME_GEN = $(BIN_DIR)/runtime-gen
bin/runtime-gen:
	$(ROOT)/hack/build.sh runtime-gen


.PHONY: bin/golangci-lint
GOLANGCI_LINT = $(BIN_DIR)/golangci-lint
bin/golangci-lint:
	# DON'T track the version of this cmd by go.mod
	$(ROOT)/hack/download.sh go_install $(GOLANGCI_LINT) github.com/golangci/golangci-lint/v2/cmd/golangci-lint v2.1.6 "version --short"

.PHONY: bin/kind
KIND = $(BIN_DIR)/kind
bin/kind:
	$(ROOT)/hack/download.sh go_install $(KIND) sigs.k8s.io/kind $(KIND_VERSION) "version | awk '{print \$$2}'"

.PHONY: bin/license-eye
LICENSE_EYE = $(BIN_DIR)/license-eye
bin/license-eye:
	if [ ! -f $(LICENSE_EYE) ]; then $(ROOT)/hack/download.sh go_install $(LICENSE_EYE) github.com/apache/skywalking-eyes/cmd/license-eye 049742de2276515409e3109ca2a91934053e080d; fi

.PHONY: bin/ginkgo
GINKGO = $(BIN_DIR)/ginkgo
bin/ginkgo:
	$(ROOT)/hack/download.sh go_install $(GINKGO) github.com/onsi/ginkgo/v2/ginkgo v2.23.3 "version | awk '{print \"v\"\$$3}'"

.PHONY: bin/mdtoc
MDTOC = $(BIN_DIR)/mdtoc
bin/mdtoc:
	$(ROOT)/hack/download.sh go_install $(MDTOC) sigs.k8s.io/mdtoc v1.1.0

.PHONY: bin/helm
HELM = $(BIN_DIR)/helm
bin/helm:
	$(ROOT)/hack/download.sh go_install $(HELM) helm.sh/helm/v3/cmd/helm v3.17.3 "version --template='{{.Version}}' | xargs printf '%s.3'"
