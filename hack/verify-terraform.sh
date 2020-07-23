#!/usr/bin/env bash

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

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source $ROOT/hack/lib.sh

hack::ensure_terraform

terraform_modules=$(find ./deploy -mindepth 1 -maxdepth 1 -type d -a -not -name 'modules')

for module in $terraform_modules; do
    echo "Checking module ${module}"
    pushd ${module} >/dev/null
    if ${TERRAFORM_BIN} fmt -check >/dev/null; then
        echo "Initialize module ${module}..."
        ${TERRAFORM_BIN} init >/dev/null
        if ! ${TERRAFORM_BIN} validate > /dev/null; then
            echo "terraform validate failed for ${module}"
            exit 1
        fi
    else
        echo "terraform fmt -check failed for ${module}"
        ${TERRAFORM_BIN} fmt
    fi
    popd >/dev/null
done

git diff --quiet deploy
