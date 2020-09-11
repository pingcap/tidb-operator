#!/bin/bash

set -e

TIMEOUT=10
COV_FILE=/tmp/e2e.coverage.txt

# Add default command if no command provided or the first argument is an
# option.
if [ $# -lt 1 -o "${1:0:1}" = '-' ]; then
    set -- /usr/local/bin/ginkgo "$@"
fi

# If google-cloud-sdk is detected, install it.
if [ -d /google-cloud-sdk ]; then
    source /google-cloud-sdk/path.bash.inc
    export CLOUDSDK_CORE_DISABLE_PROMPTS=1
fi

eval "($@) &"
PID=$!

# upload code coverage
uploadcov() {
  # The coverage profile is only written when the process exit, so
  # wait for the file to present and then upload
  for ((i=1; i<=$TIMEOUT; i++)); do
    if [ -f $COV_FILE ]; then
      echo "coverage file $COV_FILE found"
      break
    fi
    echo "coverage file not found, already waited for ${i}s"
    sleep 1
  done

  if [ ! -f $COV_FILE ]; then
    echo "coverage file not found, abort uploading"
    exit 0
  fi

  echo "uploading code coverage"
  /codecov -K \
    -B $BRANCH \
    -b $BUILD \
    -C $COMMIT \
    -P $PR \
    -t $CODECOV_TOKEN \
    -F e2e,tests -f $COV_FILE || \
    echo "codecov upload failed, ignore and continue"
}

sigterm() {
  echo "SIGTERM received, start shutdown"
  echo "stopping the process"
  kill -TERM $PID

  uploadcov
}

trap 'sigterm' TERM
wait
