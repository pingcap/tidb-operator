#!/bin/bash

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

# Pin all k8s.io dependencies to a specified version.
#
# Kubernetes staging repos are hosted in code repo (k8s.io/kubernetes). If we
# use pkgs from k8s.io/kubernetes, `replace` directive must be used to override
# version constraints derived from k8s.io/kubernetes.

VERSION=${VERSION:-${1:-}}

if [ -z "$VERSION" ]; then
    echo "VERSION is required, e.g. VERSION=x.y.z $0 or $0 x.y.z"
    exit 1
fi

echo "VERSION: $VERSION"

# Explicitly opt into go modules, even though we're inside a GOPATH directory
export GO111MODULE=on

go mod edit -require k8s.io/kubernetes@v$VERSION

#
# Return true if "$v2" is greater or equal to "$v1".
#
# Usage: version_ge "$v1" "$v2"
#
function version_ge() {
    local a="$1"
    local b="$2"
    [[ "${a}" == $(echo -e "${a}\n${b}" | sort -s -t. -k 1,1n -k 2,2n -k3,3n | head -n1) ]]
}

if version_ge "1.15.0" $VERSION; then
    STAGING_REPOS=($(curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod | sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'))
else
    STAGING_REPOS=($(curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/staging/README.md | sed -n 's|.*\[`\(k8s.io/[^`]*\)`\].*|\1|p'))
fi

edit_args=(
    -fmt
    # workaround for https://github.com/uber-go/atomic
    -replace github.com/uber-go/atomic=go.uber.org/atomic@v1.5.0
)

for repo in ${STAGING_REPOS[@]}; do
    if version_ge "1.17.0" $VERSION; then
        echo ">=1.17.0"
        staging_v=${VERSION/#1/0}
	    edit_args+=(-replace $repo=$repo@v$staging_v)
    else
	    edit_args+=(-replace $repo=$repo@kubernetes-$VERSION)
    fi
done
echo "edit_args=$edit_args"

go mod edit ${edit_args[@]}
# workaround for https://github.com/golang/go/issues/33008
# go mod tidy does not always remove unncessary lines from go.sum. For now we
# can remove it first and populate again.
rm go.sum
go mod tidy
