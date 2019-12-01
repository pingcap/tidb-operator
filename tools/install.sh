#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname $0)"
make $(make list)

# This was an attempt to convince go mode tidy to not remove everything
# However, not all binaries also provide a library
#
#echo "package tools" > tools.go
#cat go.mod | sed 's|require (|import (|' \
#    | sed 's|^\s*\(github.com\S*\)\s*v.*|    _ "\1"|' \
#    | tail --lines=+2 >> tools.go
#GO111MODULE=on go build tools.go
