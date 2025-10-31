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
KUBE_OPT = -n tidb-admin --context kind-tidb-operator

ALL_CMD = tidb-operator prestop-checker testing-workload tidb-backup-manager
.PHONY: build
build: $(addprefix build/,$(ALL_CMD))
build/%:
	$(ROOT)/hack/build.sh $*

.PHONY: image
image: $(addprefix image/,$(ALL_CMD))
image/%:
ifeq ($(DEV_MODE),true)
	@echo "Building dev image for $*..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "$(BIN_DIR)/$*" ]; then \
		echo "Binary $(BIN_DIR)/$* not found, building..."; \
		$(ROOT)/hack/build.sh $*; \
	fi
	$(ROOT)/hack/image.sh $* --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev
else
	$(ROOT)/hack/image.sh $*
endif

.PHONY: push
push: $(addprefix push/,$(ALL_CMD))
push/%:
ifeq ($(DEV_MODE),true)
	@echo "Building and pushing dev image for $*..."
	@$(MAKE) dev-prepare-output
	@if [ ! -f "$(BIN_DIR)/$*" ]; then \
		echo "Binary $(BIN_DIR)/$* not found, building..."; \
		$(ROOT)/hack/build.sh $*; \
	fi
	$(ROOT)/hack/image.sh $* --dockerfile=Dockerfile.dev --dockerignore=.dockerignore.dev --push
else
	$(ROOT)/hack/image.sh $* --push
endif

# Ensure _output directory is a real directory with binaries, not a symlink
.PHONY: dev-prepare-output
dev-prepare-output:
	@if [ -L "$(OUTPUT_DIR)" ]; then \
		REAL_PATH=$$(readlink "$(OUTPUT_DIR)"); \
		echo "Converting $(OUTPUT_DIR) symlink to real directory for Docker build..."; \
		rm "$(OUTPUT_DIR)"; \
		mkdir -p "$(OUTPUT_DIR)"; \
		if [ -d "$$REAL_PATH" ]; then \
			cp -r "$$REAL_PATH"/* "$(OUTPUT_DIR)"/; \
		fi; \
	elif [ ! -d "$(OUTPUT_DIR)" ]; then \
		echo "Creating $(OUTPUT_DIR) directory..."; \
		mkdir -p "$(OUTPUT_DIR)"; \
	fi

.PHONY: deploy
deploy: bin/kubectl release
	$(KUBECTL) $(KUBE_OPT) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator.crds.yaml
	$(KUBECTL) $(KUBE_OPT) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator.yaml

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

ALL_GEN = tidy codegen crd runtimegen gengo overlaygen doc
.PHONY: generate
generate: $(ALL_GEN) license

.PHONY: verify/license
verify/license: bin/license-eye
	$(LICENSE_EYE) -c .github/licenserc.yaml header check

.PHONY: verify/feature-gates
verify/feature-gates:
	cd $(ROOT) && go run cmd/verify-feature-gates/main.go

.PHONY: verify
verify: $(addprefix verify/,$(ALL_GEN)) verify/license verify/feature-gates
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
e2e/deploy: bin/kubectl release
	$(KUBECTL) $(KUBE_OPT) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator.crds.yaml
	$(KUBECTL) $(KUBE_OPT) apply --server-side=true -f $(OUTPUT_DIR)/manifests/tidb-operator-e2e.yaml

.PHONY: kube
kube: bin/kind bin/kubectl
	@echo "ensure that the kubernetes env is existing"
	V_KIND=$(KIND) V_KUBECTL=$(KUBECTL) $(ROOT)/hack/kind.sh

.PHONY: reload/operator
reload/operator: bin/kubectl
	$(KUBECTL) $(KUBE_OPT) delete pod `$(KUBECTL) $(KUBE_OPT) get pods | awk '/operator/{ print $$1 }'`

.PHONY: logs/operator
logs/operator: bin/kubectl
	$(KUBECTL) $(KUBE_OPT) logs -f `$(KUBECTL) $(KUBE_OPT) get pods | awk '/operator/{ print $$1 }'`

OVERLAY_GEN = $(BIN_DIR)/overlay-gen
bin/overlay-gen:
	$(ROOT)/hack/build.sh overlay-gen

RUNTIME_GEN = $(BIN_DIR)/runtime-gen
bin/runtime-gen:
	$(ROOT)/hack/build.sh runtime-gen

ALL_BIN = kubectl golangci-lint kind ginkgo mdtoc helm license-eye mockgen controller-gen deepcopy-gen register-gen

# Generic target for allowed bin/xxx tools - automatically defines XXX variable (with hyphens converted to underscores)
# e.g. bin/abc-def will define ABC_DEF = $(BIN_DIR)/abc-def
define make_bin_target
$(eval $(shell echo $(1) | tr '[:lower:]-' '[:upper:]_') = $(BIN_DIR)/$(1))
endef

.PHONY: $(addprefix bin/,$(ALL_BIN))
$(addprefix bin/,$(ALL_BIN)):
	$(call make_bin_target,$(patsubst bin/%,%,$@))
	./hack/tools.sh $($(shell echo $(patsubst bin/%,%,$@) | tr '[:lower:]-' '[:upper:]_'))

