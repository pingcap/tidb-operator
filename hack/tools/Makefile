# Copyright 2020 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

export GO111MODULE=on

OUTPUT_DIR ?= bin
DEP_PROGS=\
	misspell\
	goimports\
	golangci-lint\
	controller-gen

.PHONY: all
all: clean build

.PHONY: clean
clean:
	$(foreach cmd, $(DEP_PROGS), rm -f $(OUTPUT_DIR)/$(cmd)) \
	rm -rf vendor/

.PHONY: build
build: $(DEP_PROGS)

.PHONY: deps
deps:
	go mod vendor

golangci-lint: CMD_DIR ?= $(shell go list -f "{{ .Dir }}" github.com/golangci/golangci-lint/cmd/golangci-lint)
golangci-lint: deps
	go build -o $(OUTPUT_DIR)/golangci-lint $(CMD_DIR)

misspell: CMD_DIR ?= $(shell go list -f "{{ .Dir }}" github.com/client9/misspell/cmd/misspell)
misspell: deps
	go build -o $(OUTPUT_DIR)/misspell $(CMD_DIR)

goimports: CMD_DIR ?= $(shell go list -f "{{ .Dir }}" golang.org/x/tools/cmd/goimports)
goimports: deps
	go build -o $(OUTPUT_DIR)/goimports $(CMD_DIR)

controller-gen: CMD_DIR ?= $(shell go list -f "{{ .Dir }}" sigs.k8s.io/controller-tools/cmd/controller-gen)
controller-gen: deps
	go build -o $(OUTPUT_DIR)/controller-gen $(CMD_DIR)
