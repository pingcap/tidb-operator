#!/usr/bin/env bash

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

function license-eye() {
    download go_install $1 \
        github.com/apache/skywalking-eyes/cmd/license-eye \
        049742de2276515409e3109ca2a91934053e080d
}

function mockgen() {
    download go_install $1 \
        go.uber.org/mock/mockgen \
        v0.5.0
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
