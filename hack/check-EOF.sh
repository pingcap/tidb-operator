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
    .github
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
        echo "$f"
        NUM=$((NUM + 1))
    fi
done

if [ $NUM -ne 0 ]; then
    echo Has file not end with newline.
    exit 1
fi
