#!/usr/bin/env bash
#
# Example (but you can omit the last argument):
#
#   ./add-tool.sh github.com/jirfag/go-queryset master cmd/goqueryset 
set -euo pipefail

REPO="$1"
repo_name="$(basename $1)"
VERSION="${2:-master}"
PKG="${3:-""}"

if [[ -z $PKG ]]; then
  NAME="$repo_name"
else
  PKG="/${PKG}"
  NAME="$(basename "${PKG}")"
fi

FULL="${REPO}${PKG}"

cat << EOF >> Makefile

bin/${NAME}: go.mod
	\$(GO) get -d ${REPO}@${VERSION}; \\
	\$(GO) build -o bin/${NAME} ${FULL};
EOF
