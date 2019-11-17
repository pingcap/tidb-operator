#!/usr/bin/env bash

# Copyright 2019 PingCAP, Inc.
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

set -o errexit
set -o nounset
set -o pipefail

# generate-internal-groups generates api registries for api groups

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename "$0") <apis-package> ...

  <apis-package>      the internal types dir (e.g. github.com/example/project/pkg/apis).
  ...                 arbitrary flags passed to generator bin

Examples:
  $(basename "$0") github.com/example/project/pkg/apis
EOF
  exit 0
fi

function codegen::join() { local IFS="$1"; shift; echo "$*"; }

APIS_PKG="$1"
shift 1

export GO111MODULE=on

go install sigs.k8s.io/apiserver-builder-alpha/cmd/apiregister-gen

APIS_DIR=("${APIS_PKG}/...")

echo "Generating api registries"
"${GOPATH}/bin/apiregister-gen" --input-dirs "$(codegen::join , "${APIS_DIR[@]}")" "$@"
