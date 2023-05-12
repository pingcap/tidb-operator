#!/usr/bin/env bash

# Copyright 2023 PingCAP, Inc.
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

# This script is similar with `./hack/version.sh`, but used to customize ports when building.

set -euo pipefail

function tidb_operator::port::ldflag() {
  local key=${1}
  local val=${2}

  echo "-X 'github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.${key}=${val}'"
}

# Prints the value that needs to be passed to the -ldflags parameter of go build
function tidb_operator::port::ldflags() {
  local -a ldflags=()

  if [[ -n ${TIDB_SERVER_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiDBServer" "${TIDB_SERVER_PORT}"))
  fi
  if [[ -n ${TIDB_STATUS_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiDBStatus" "${TIDB_STATUS_PORT}"))
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}

# output -ldflags parameters
tidb_operator::port::ldflags
