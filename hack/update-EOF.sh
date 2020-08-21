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

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

cd $ROOTDIR

FILELIST=($(find . -type f -not \( -path './vendor/*' \
    -o -path './output/*' \
    -o -path './_tools/*' \
    -o -path './.git/*' \
    -o -path './*/.terraform/*' \
    -o -path './images/*/bin/*' \
    -o -path './tests/images/*/bin/*' \
    -o -path '*.png' \
    -o -path './tkctl' \
    -o -path './.idea/*' \
    -o -path './.DS_Store' \
    -o -path './*/.DS_Store' \
    \)))

for f in ${FILELIST[@]}; do
    c=$(tail -c 1 "$f" | wc -l)
    if [ "$c" -eq 0 ]; then
        echo "find file $f do not end with newline, fixing it"
        printf "\n" >> $f
    fi
done
