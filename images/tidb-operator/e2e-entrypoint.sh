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

# This is a wrapper script used as the entrypoint in E2E tests to:
# - generate a coverage report for binaries when terminating.
# - upload the coverage report to codecov

TIMEOUT=30
COV_FILE=/tmp/coverage.txt

# populate test tags
COV_TAGS="e2e"

COV_NAME="tidb-operator.$(date +%s)"
if [ -n "$COMPONENT" ]; then
    COV_NAME="$COMPONENT.$(date +%s)"
fi

# run the original process in the background, so that we can send SIGTERM to it.
eval "($@) &"
PID=$!

# upload the coverage report to codecov after generated after the original process exited normally.
upload_cov() {
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

  echo "some env for uploading code coverage"
  echo "SRC_BRANCH: $SRC_BRANCH"
  echo "BUILD_NUMBER: $BUILD_NUMBER"
  echo "GIT_COMMIT: $GIT_COMMIT"
  echo "PR_ID: $PR_ID"
  echo "COV_NAME: $COV_NAME"

  echo "e2e-entrypoint.sh: uploading code coverage"
  /codecov \
    -t "$CODECOV_TOKEN" \
    -B "$SRC_BRANCH" \
    -b "$BUILD_NUMBER" \
    -C "$GIT_COMMIT" \
    -P "$PR_ID" \
    -F "$COV_TAGS" \
    -n "$COV_NAME" \
    -f "$COV_FILE" || \
    echo "e2e-entrypoint.sh: codecov upload failed, ignore and continue"
}

_term() {
  echo "e2e-entrypoint.sh: SIGTERM received, start to shutdown"
  echo "e2e-entrypoint.sh: stopping the process"
  kill -TERM $PID

  upload_cov
}

trap _term SIGTERM

wait
