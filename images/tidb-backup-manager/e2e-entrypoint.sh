#!/bin/sh

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

# This is a wrapper script used as the entrypoint in E2E tests to:
# - generate a coverage report for binaries when terminating.
# - upload the coverage report to codecov

TIMEOUT=30
COV_FILE=/tmp/coverage.txt

# populate test tags
COV_TAGS="e2e"
if [ -n "$COMPONENT" ]; then
    COV_TAGS="$COV_TAGS,$COMPONENT"
fi

COV_NAME="tidb-operator.$(date +%s)"
if [ -n "$COMPONENT" ]; then
    COV_NAME="$COMPONENT.$(date +%s)"
fi

export GOOGLE_APPLICATION_CREDENTIALS=/tmp/google-credentials.json
echo "Create rclone.conf file."
cat <<EOF > /tmp/rclone.conf
[s3]
type = s3
env_auth = true
provider =  ${S3_PROVIDER}
access_key_id = ${AWS_ACCESS_KEY_ID}
secret_access_key = ${AWS_SECRET_ACCESS_KEY:-$AWS_SECRET_KEY}
region = ${AWS_REGION}
acl = ${AWS_ACL}
endpoint = ${S3_ENDPOINT}
storage_class = ${AWS_STORAGE_CLASS}
[gcs]
type = google cloud storage
project_number = ${GCS_PROJECT_ID}
service_account_file = ${GOOGLE_APPLICATION_CREDENTIALS}
object_acl = ${GCS_OBJECT_ACL}
bucket_acl = ${GCS_BUCKET_ACL}
location =  ${GCS_LOCATION}
storage_class = ${GCS_STORAGE_CLASS:-"COLDLINE"}
[azure]
type = azureblob
account = ${AZUREBLOB_ACCOUNT}
key = ${AZUREBLOB_KEY}
EOF

if [[ -n "${GCS_SERVICE_ACCOUNT_JSON_KEY:-}" ]]; then
    echo "Create google-credentials.json file."
    cat <<EOF > ${GOOGLE_APPLICATION_CREDENTIALS}
    ${GCS_SERVICE_ACCOUNT_JSON_KEY}
EOF
else
    touch ${GOOGLE_APPLICATION_CREDENTIALS}
fi

BACKUP_BIN=/tidb-backup-manager

# exec command
case "$1" in
    backup)
        shift 1
        echo "$BACKUP_BIN backup $@"
        $BACKUP_BIN backup "$@" &
        ;;
    export)
        shift 1
        echo "$BACKUP_BIN export $@"
        $BACKUP_BIN export "$@" &
        ;;
    restore)
        shift 1
        echo "$BACKUP_BIN restore $@"
        $BACKUP_BIN restore "$@" &
        ;;
    import)
        shift 1
        echo "$BACKUP_BIN import $@"
        $BACKUP_BIN import "$@" &
        ;;
    clean)
        shift 1
        echo "$BACKUP_BIN clean $@"
        $BACKUP_BIN clean "$@" &
        ;;
    *)
        echo "Usage: $0 {backup|restore|clean}"
        echo "Now runs your command."
        echo "$@"

        exec "$@"
esac

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

  echo "e2e-entrypoint.sh: uploading code coverage"
  /codecov \
    -t "$CODECOV_TOKEN" \
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
