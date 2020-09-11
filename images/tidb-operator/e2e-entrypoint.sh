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


set -e

# This is an wrapper script used as entrypoint in E2E tests

TIMEOUT=30
COV_FILE=/tmp/e2e.coverage.txt

# populate test tags
COV_TAGS="e2e"
if [ ! -z $COMPONENT ]; then
    COV_TAGS="$COV_TAGS,$COMPONENT"
fi

eval "($@) &"
PID=$!

# upload code coverage
uploadcov() {
  # The coverage profile is only written when the process exit, so
  # wait for the file to present and then upload
  for ((i=1; i<=$TIMEOUT; i++)); do
    if [ -f $COV_FILE ]; then
      echo "e2e-entrypoint.sh: coverage file $COV_FILE found"
      break
    fi
    echo "e2e-entrypoint.sh: coverage file not found, already waited for ${i}s"
    sleep 1
  done

  if [ ! -f $COV_FILE ]; then
    echo "e2e-entrypoint.sh: coverage file not found, abort uploading"
    exit 0
  fi

  echo "e2e-entrypoint.sh: uploading code coverage"
  /codecov -K \
    -B $BRANCH \
    -b $BUILD.$(date +%s) \
    -C $COMMIT \
    -P $PR \
    -t $CODECOV_TOKEN \
    -F $COV_TAGS -f $COV_FILE || \
    echo "e2e-entrypoint.sh: codecov upload failed, ignore and continue"
}

sigterm() {
  echo "e2e-entrypoint.sh: SIGTERM received, start shutdown"
  echo "e2e-entrypoint.sh: stopping the process"
  kill -TERM $PID

  uploadcov
}

trap 'sigterm' TERM
wait
