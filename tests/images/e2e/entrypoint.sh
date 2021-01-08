#!/bin/bash

set -e

# Add default command if no command provided or the first argument is an
# option.
if [ $# -lt 1 -o "${1:0:1}" = '-' ]; then
    # this will replace $1 with /usr/local/bin/ginkgo, and any other arguments shift by one
    # ref: https://unix.stackexchange.com/a/308263/418182
    set -- /usr/local/bin/ginkgo "$@"
fi

# If google-cloud-sdk is detected, install it.
if [ -d /google-cloud-sdk ]; then
    source /google-cloud-sdk/path.bash.inc
    export CLOUDSDK_CORE_DISABLE_PROMPTS=1
fi

exec "$@"
