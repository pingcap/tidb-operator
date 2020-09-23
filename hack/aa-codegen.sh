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

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

${scriptdir}/generate-internal-groups.sh \
  all \
  "tidb.pingcap.com:v1alpha1" \
  --go-header-file=${scriptdir}/boilerplate/boilerplate.generatego.txt
