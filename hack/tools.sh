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

# DEPRECATED
# It's not called now. All tools are migrated to use tools feature introduced in go1.24
# Just keep it if some other tools which are not written by go are introduced.

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/..; pwd -P)

source $ROOT/hack/lib/vars.sh
source $ROOT/hack/lib/download.sh

# Just keep as example
function kubectl() {
    download s_curl $1 \
        https://dl.k8s.io/release/VERSION/bin/OS/ARCH/kubectl \
        v1.31.0 \
        "version --client | awk '/Client/{print \$3}'"
}

# Create a script to call go tool in _output/bin
function go_tools() {
    echo "installing ${1}"
	cat << EOF > ${ROOT}/_output/bin/${1}
#!/usr/bin/env bash
go tool -modfile ${ROOT}/tools/${1}/go.mod ${1} "\${@}"
EOF

    chmod +x ${ROOT}/_output/bin/${1}
}


go_tools $1
