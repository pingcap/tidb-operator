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

source "${ROOT}/hack/lib.sh"

FILE_SERVER_URL=http://fileserver.pingcap.net
FILE_BASE_DIR=pingcap/tidb-operator
GITHASH=${GITHASH:-}
BUILD_BRANCH=${BUILD_BRANCH:-}

if [ -z "$GITHASH" -o -z "$BUILD_BRANCH" ]; then
    echo "error: GITHASH/BUILD_BRANCH are required"
    exit 1
fi

function upload() {
    local dir=$(mktemp -d)
    trap "test -d $dir && rm -rf $dir" RETURN

    echo "info: create a temporary directory: $dir"

    echo "info: uploading charts and binaries"
    tar -zcvf $dir/tidb-operator.tar.gz images/tidb-operator images/tidb-backup-manager charts
    curl -F ${FILE_BASE_DIR}/builds/${GITHASH}/tidb-operator.tar.gz=@$dir/tidb-operator.tar.gz  ${FILE_SERVER_URL}/upload

    echo "info: update ref of branch '$BUILD_BRANCH'"
    echo -n $GITHASH > $dir/sha1
    curl -F ${FILE_BASE_DIR}/refs/${BUILD_BRANCH}/sha1=@$dir/sha1  ${FILE_SERVER_URL}/upload
}

# retry a few times until it succeeds, this can avoid temporary network flakes
hack::wait_for_success 120 5 "upload"
