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

GOBIN ?= $(shell go env GOPATH)/bin
DEP_PROGS=\
	$(GOBIN)/misspell\
	$(GOBIN)/goimports\
	$(GOBIN)/golangci-lint

.PHONY: all
all: clean build

.PHONY: clean
clean:
	rm -f $(DEP_PROGS)
	rm -rf vendor/

.PHONY: build
build: $(DEP_PROGS)

.PHONY: deps
deps:
	go mod vendor

$(GOBIN)/golangci-lint: deps
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

$(GOBIN)/misspell: deps
	go install github.com/client9/misspell/cmd/misspell

$(GOBIN)/goimports: deps
	go install golang.org/x/tools/cmd/goimports
