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
GO_MODULE := github.com/pingcap/tidb-operator
OVERLAY_PKG_DIR = $(ROOT)/pkg/overlay
BOILERPLATE_FILE = $(ROOT)/hack/boilerplate/boilerplate.go.txt
MOCK_BOILERPLATE_FILE = $(ROOT)/hack/boilerplate/boilerplate.txt

KIND_VERSION ?= v0.24.0

# TODO: use kubectl in _output
KUBECTL = kubectl -n tidb-admin --context kind-tidb-operator

ALL_CMD = operator prestop-checker testing-workload backup-manager
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

.PHONY: deploy
deploy: crd
	$(KUBECTL) apply --server-side=true -f $(ROOT)/manifests/crd
	$(KUBECTL) apply --server-side=true -f $(ROOT)/manifests/rbac
	$(KUBECTL) apply --server-side=true -f $(ROOT)/manifests/deploy

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

.PHONY: crd
crd: bin/controller-gen
	$(CONTROLLER_GEN) crd:generateEmbeddedObjectMeta=true output:crd:artifacts:config=$(ROOT)/manifests/crd paths=$(API_PATH)/...

# Deprecate this generator, rbac generator cannot well handle nonResourceURLs
.PHONY: rbac
rbac: bin/controller-gen
	$(CONTROLLER_GEN) rbac:roleName=tidb-operator output:rbac:artifacts:config=$(ROOT)/manifests/rbac paths=$(API_PATH)/...

.PHONY: tidy
tidy:
	cd $(API_PATH) && go mod tidy
	go mod tidy

gengo: GEN_DIR ?= ./...
gengo: bin/mockgen
	BOILERPLATE_FILE=${MOCK_BOILERPLATE_FILE} GOBIN=$(BIN_DIR) GO_MODULE=$(GO_MODULE) go generate $(GEN_DIR)

.PHONY: license
license: bin/license-eye
	$(LICENSE_EYE) -c .github/licenserc.yaml header fix

ALL_GEN = tidy codegen crd gengo overlaygen
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
	$(GOLANGCI_LINT) run ./...

.PHONY: lint-fix
lint-fix: bin/golangci-lint
	$(GOLANGCI_LINT) run ./... --fix

.PHONY: unit
unit:
	go test $$(go list -e ./... | grep -v cmd | grep -v tools | grep -v tests | grep -v third_party) \
		-cover -coverprofile=coverage.txt -covermode=atomic
	sed -i.bak '/generated/d;/fake.go/d' coverage.txt && rm coverage.txt.bak

.PHONY: check
check: lint unit verify


.PHONY: e2e/prepare
e2e/prepare: bin/kind crd
	$(ROOT)/hack/e2e.sh --prepare

.PHONY: e2e/run
e2e/run:
	$(ROOT)/hack/e2e.sh run $(GINKGO_OPTS)

.PHONY: e2e
e2e: bin/kind crd
	$(ROOT)/hack/e2e.sh --prepare run $(GINKGO_OPTS)

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
	$(ROOT)/hack/download.sh go_install $(CONTROLLER_GEN) sigs.k8s.io/controller-tools/cmd/controller-gen

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


.PHONY: bin/golangci-lint
GOLANGCI_LINT = $(BIN_DIR)/golangci-lint
bin/golangci-lint:
	# DON'T track the version of this cmd by go.mod
	$(ROOT)/hack/download.sh go_install $(GOLANGCI_LINT) github.com/golangci/golangci-lint/cmd/golangci-lint v1.62.2 "version --format=short"

.PHONY: bin/kind
KIND = $(BIN_DIR)/kind
bin/kind:
	$(ROOT)/hack/download.sh go_install $(KIND) sigs.k8s.io/kind $(KIND_VERSION) "version | awk '{print \$$2}'"

.PHONY: bin/license-eye
LICENSE_EYE = $(BIN_DIR)/license-eye
bin/license-eye:
	if [ ! -f $(LICENSE_EYE) ]; then $(ROOT)/hack/download.sh go_install $(LICENSE_EYE) github.com/apache/skywalking-eyes/cmd/license-eye v0.6.0; fi

