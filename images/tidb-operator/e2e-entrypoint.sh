#!/bin/bash

# Copyright 2021 PingCAP, Inc.
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

set -e

# This is a wrapper script used as the entrypoint in E2E tests to
# replace the coverage filename with a time based part.

COV_NAME="tidb-operator.$(( ( RANDOM % 100000 ) + 1 ))"
if [ -n "$COMPONENT" ]; then
    COV_NAME="$COMPONENT.$(( ( RANDOM % 100000 ) + 1 ))"
fi

# add a time part into coverage filename.
args=( "$@" )
for ((i=0; i < $#; i++)); do
  if [[ "${args[$i]}" =~ ^-test.coverprofile ]]; then
    args[$i]="-test.coverprofile=/coverage/$COV_NAME.cov"
    break
  fi
done
set "${args[@]}"

eval "($@) &"
PID=$!

_term() {
  kill -TERM $PID
  wait $PID
  sleep 10
}

trap _term SIGTERM
wait $PID
