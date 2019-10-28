#!/bin/bash
#
# Pin all k8s.io dependencies to a specified version.
#

VERSION=1.12.5

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
)
for repo in ${STAGING_REPOS[@]}; do
	edit_args+=(-replace $repo=$repo@kubernetes-$VERSION)
done

go mod edit ${edit_args[@]}
# workaround for https://github.com/golang/go/issues/33008
# go mod tidy does not always remove unncessary lines from go.sum. For now we
# can remove it first and populate again.
rm go.sum
go mod tidy
