#!/bin/bash
#
# This script is used to verify links in markdown docs.
#

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

if ! which markdown-link-check &>/dev/null; then
    sudo npm install -g markdown-link-check@3.8.0
fi

VERBOSE=${VERBOSE:-}
CONFIG_TMP=$(mktemp)
ERROR_REPORT=$(mktemp)

trap 'rm -f $CONFIG_TMP $ERROR_REPORT' EXIT

function in_array() {
    local i=$1
    shift
    local a=("${@}")
    local e
    for e in "${a[@]}"; do
        [[ "$e" == "$i" ]] && return 0;
    done
    return 1
}

# Check all directories starting with 'v\d.*' and dev.
for d in zh en; do
    echo "info: checking links under $ROOT/$d directory..."
    sed \
        -e "s#<ROOT>#$ROOT#g" \
        $ROOT/hack/markdown-link-check.tpl > $CONFIG_TMP
    if [ -n "$VERBOSE" ]; then
        cat $CONFIG_TMP
    fi
    while read -r tasks; do
        for task in $tasks; do
            (
                output=$(markdown-link-check --color --config "$CONFIG_TMP" "$task" -q)
                if [ $? -ne 0 ]; then
                    printf "$output" >> $ERROR_REPORT
                fi
                if [ -n "$VERBOSE" ]; then
                    echo "$output"
                fi
            ) &
        done
        wait
    done <<<"$(find "$ROOT/$d" -type f -not -path './node_modules/*' -name '*.md' | xargs -n 10)"
done

error_files=$(cat $ERROR_REPORT | grep 'FILE: ' | wc -l)
error_output=$(cat $ERROR_REPORT)
echo ""
if [ "$error_files" -gt 0 ]; then
    echo "Link error: $error_files files have invalid links. The faulty files are listed below, please fix the wrong links!"
    echo ""
    echo "=== ERROR REPORT == ":
    echo "$error_output"
    exit 1
else
    echo "info: all files are ok!"
fi
