#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"

CMD="/usr/bin/find $ROOTDIR/"

EXCLUDE_DIR=(
    vendor
    output
    _tools
    .git
    static/tidb-operator-overview.png
)

for DIR in ${EXCLUDE_DIR[@]}; do
    CMD="$CMD -path $ROOTDIR/$DIR -prune -o"
done

FILELIST=$($CMD -type f -print)

check() {
    x=$(tail -c 1 "$1")
    if [ "$x" != "" ]; then
        echo "$1"
    fi
}

NUM=0
for f in ${FILELIST[@]}; do
    c=$(tail -c 1 "$f")
    if [ "$c" != "" ]; then
        failed_file+=($f)
        NUM=$((NUM + 1))
    fi
done

if [ $NUM -ne 0 ]; then
    echo "error: following files do not end with newline, please fix them"
    printf '%s\n' "${failed_file[@]}"
    exit 1
else
    echo "all files pass checking."
fi
