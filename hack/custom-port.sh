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

  if [[ -n ${PD_CLIENT_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortPDClient" "${PD_CLIENT_PORT}"))
  fi
  if [[ -n ${PD_PEER_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortPDPeer" "${PD_PEER_PORT}"))
  fi

  if [[ -n ${TIKV_SERVER_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiKVServer" "${TIKV_SERVER_PORT}"))
  fi
  if [[ -n ${TIKV_STATUS_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiKVStatus" "${TIKV_STATUS_PORT}"))
  fi

  if [[ -n ${TIFLASH_TCP_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiFlashTcp" "${TIFLASH_TCP_PORT}"))
  fi
  if [[ -n ${TIFLASH_HTTP_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiFlashHttp" "${TIFLASH_HTTP_PORT}"))
  fi
  if [[ -n ${TIFLASH_FLASH_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiFlashFlash" "${TIFLASH_FLASH_PORT}"))
  fi
  if [[ -n ${TIFLASH_PROXY_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiFlashProxy" "${TIFLASH_PROXY_PORT}"))
  fi
  if [[ -n ${TIFLASH_METRICS_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiFlashMetrics" "${TIFLASH_METRICS_PORT}"))
  fi
  if [[ -n ${TIFLASH_PROXY_STATUS_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiFlashProxyStatus" "${TIFLASH_PROXY_STATUS_PORT}"))
  fi
  if [[ -n ${TIFLASH_INTERNAL_PORT-} ]]; then
  ldflags+=($(tidb_operator::port::ldflag "customPortTiFlashInternal" "${TIFLASH_INTERNAL_PORT}"))
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}

# output -ldflags parameters
tidb_operator::port::ldflags
