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

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

boiler="${ROOT}/hack/boilerplate/boilerplate.py"

#
# TODO update license information for following files
#
# - ./deploy/*
# - */*.sh
# - */Makefile
# - */Dockerfile
# - */*.register.go # files geneated by apiregister-gen
#
files=($(find . -type f -not \( \
        -path './hack/boilerplate/*' \
        -o -path './_tools/*' \
        -o -path './.git/*' \
        -o -path './.*/*' \
        -o -path './vendor/*' \
        -o -path './pkg/client/*' \
        -o -path './*/.terraform/*' \
        -o -path './deploy/*' \
        -o -path '*/*.sh' \
        -o -path '*/Makefile' \
        -o -path '*/Dockerfile' \
        -o -path '*/*.register.go' \
    \)
))

files_need_boilerplate=()
while IFS=$'\n' read -r line; do
  files_need_boilerplate+=( "$line" )
done < <("${boiler}" "${files[@]}")

# Run boilerplate check
if [[ ${#files_need_boilerplate[@]} -gt 0 ]]; then
    for file in "${files_need_boilerplate[@]}"; do
        echo "Boilerplate header is wrong for: ${file}" >&2
    done
    exit 1
fi
