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

UCLOUD_PUBLIC_KEY=${UCLOUD_PUBLIC_KEY:-}
UCLOUD_PRIVATE_KEY=${UCLOUD_PRIVATE_KEY:-}
UCLOUD_UFILE_PROXY_HOST=${UCLOUD_UFILE_PROXY_HOST:-}
UCLOUD_UFILE_API_HOST=${UCLOUD_UFILE_API_HOST:-api.spark.ucloud.cn}
UCLOUD_UFILE_BUCKET=${UCLOUD_UFILE_BUCKET:-}
GITHASH=${GITHASH:-}
BUILD_BRANCH=${BUILD_BRANCH:-}

FILEMGR_URL="http://tools.ufile.ucloud.com.cn/filemgr-linux64.tar.gz"

if [ -z "$UCLOUD_PUBLIC_KEY" -o -z "$UCLOUD_PRIVATE_KEY" -o -z "$UCLOUD_UFILE_PROXY_HOST" ]; then
    echo "error: UCLOUD_PUBLIC_KEY/UCLOUD_PUBLIC_KEY/UCLOUD_UFILE_PROXY_HOST are required"
    exit 1
fi

if [ -z "$UCLOUD_UFILE_BUCKET" ]; then
    echo "error: UCLOUD_UFILE_BUCKET is required"
    exit 1
fi

if [ -z "$GITHASH" -o -z "$BUILD_BRANCH" ]; then
    echo "error: GITHASH/BUILD_BRANCH are required"
    exit 1
fi

function upload() {
    local dir=$(mktemp -d)
    trap "test -d $dir && rm -rf $dir" RETURN

    echo "info: create a temporary directory: $dir"

    cat <<EOF > $dir/config.cfg
{
    "public_key" : "${UCLOUD_PUBLIC_KEY}",
    "private_key" : "${UCLOUD_PRIVATE_KEY}",
    "proxy_host" : "${UCLOUD_UFILE_PROXY_HOST}",
    "api_host" : "${UCLOUD_UFILE_API_HOST}"
}
EOF

    echo "info: downloading filemgr from $FILEMGR_URL"
    curl --retry 10 -L -s "$FILEMGR_URL" | tar --strip-components 2 -C $dir -xzvf - ./linux64/filemgr-linux64

    echo "info: uploading charts and binaries"
    tar -zcvf $dir/tidb-operator.tar.gz images/tidb-operator images/tidb-backup-manager charts
    $dir/filemgr-linux64 --config $dir/config.cfg --action mput --bucket ${UCLOUD_UFILE_BUCKET} --nobar --key builds/pingcap/operator/${GITHASH}/centos7/tidb-operator.tar.gz --file $dir/tidb-operator.tar.gz

    echo "info: update ref of branch '$BUILD_BRANCH'"
    echo -n $GITHASH > $dir/sha1
	$dir/filemgr-linux64 --config $dir/config.cfg --action mput --bucket ${UCLOUD_UFILE_BUCKET} --nobar --key refs/pingcap/operator/${BUILD_BRANCH}/centos7/sha1 --file $dir/sha1
}

# retry a few times until it succeeds, this can avoid temporary network flakes
hack::wait_for_success 120 5 "upload"
