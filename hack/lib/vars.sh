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

# source only once
[[ $(type -t vars::loaded) == function ]] && return 0

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/../..; pwd -P)

# Set LC_ALL to avoid sort issue
# See https://www.gnu.org/software/coreutils/manual/html_node/ls-invocation.html#ls-invocation
readonly LC_ALL=C

# ---
# Global variables definitions.
# All variables should be in format V_XXX
# ---

# V_ARCH defines architecture, which is used to build and test
# - amd64
# - arm64
readonly V_ARCH=$(go env GOARCH)

# V_OS defines operating system, which is used to build and test
readonly V_OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# V_PLATFORMS defines platforms of build binary, values can be splited by comma
# If empty, host platform will be used automatically
# - linux/amd64
# - linux/arm64
readonly V_PLATFORMS=${V_PLATFORMS:-""}

# V_IMG_HUB defines image hub of image
# - kind
readonly V_IMG_HUB=${V_IMG_HUB:-"kind"}

# V_IMG_PROJECT defines default project name of image
readonly V_IMG_PROJECT=${V_IMG_PROJECT:-"pingcap"}

# V_KIND_CLUSTER defines default cluster name of kind, the default value is tidb-operator
readonly V_KIND_CLUSTER=${V_KIND_CLUSTER:-"tidb-operator"}

# V_DEPLOY_NAMESPACE defines namespace of deploy, the default value is tidb-admin
readonly V_DEPLOY_NAMESPACE=${V_DEPLOY_NAMESPACE:-"tidb-admin"}

# V_TIDB_CLUSTER_VERSION defines version of TiDB cluster, the default value is v8.2.0
readonly V_TIDB_CLUSTER_VERSION=${V_TIDB_CLUSTER_VERSION:-"v8.5.2"}

# V_TIDB_CLUSTER_VERSION_PREV defines a older version of TiDB cluster, the default value is v8.1.0
readonly V_TIDB_CLUSTER_VERSION_PREV=${V_TIDB_CLUSTER_VERSION_PREV:-"v8.5.1"}

# V_KUBE_VERSION defines default test version of kubernetes
readonly V_KUBE_VERSION=${V_KUBE_VERSION:-"v1.31.0"}

# V_RELEASE defines the release version of tidb-operator
readonly V_RELEASE=${V_RELEASE:-"latest"}


# marker function
function vars::loaded() {
  return 0
}
