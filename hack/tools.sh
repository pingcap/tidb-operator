#!/usr/bin/env bash
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


set -o errexit
set -o nounset
set -o pipefail

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/..; pwd -P)

source $ROOT/hack/lib/vars.sh
source $ROOT/hack/lib/download.sh

function kubectl() {
    download s_curl $1 \
        https://dl.k8s.io/release/VERSION/bin/OS/ARCH/kubectl \
        v1.31.0 \
        "version --client | awk '/Client/{print \$3}'"
}

function golangci-lint () {
    # DON'T track the version of this cmd by go.mod
    download go_install $1 \
        github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
        v2.1.6 \
        "version --short"
}

function kind() {
    # DON'T track the version of this cmd by go.mod
    download go_install $1 \
        sigs.k8s.io/kind \
        v0.24.0 \
        "version | awk '{print \$2}'"
}

function ginkgo() {
    download go_install $1 \
        github.com/onsi/ginkgo/v2/ginkgo \
        v2.23.3 \
        "version | awk '{print \"v\"\$3}'"
}

function mdtoc() {
    download go_install $1 \
        sigs.k8s.io/mdtoc \
        v1.1.0
}

function helm() {
    download go_install $1 \
        helm.sh/helm/v3/cmd/helm \
        v3.17.3 \
        "version --template='{{.Version}}' | xargs printf '%s.3'"
}

function license-eye() {
    download go_install $1 \
        github.com/apache/skywalking-eyes/cmd/license-eye \
        049742de2276515409e3109ca2a91934053e080d
}

function mockgen() {
    download go_install $1 \
        go.uber.org/mock/mockgen \
        v0.6.0
}

function controller-gen() {
    download go_install $1 \
        sigs.k8s.io/controller-tools/cmd/controller-gen \
        v0.17.2 \
        "--version | awk '{print \$2}'"
}

function deepcopy-gen() {
    # version is tracked by go.mod
    download go_install $1 \
        k8s.io/code-generator/cmd/deepcopy-gen
}

function register-gen() {
    # version is tracked by go.mod
    download go_install $1 \
        k8s.io/code-generator/cmd/register-gen
}


tool=$(basename "$1")
$tool $1
