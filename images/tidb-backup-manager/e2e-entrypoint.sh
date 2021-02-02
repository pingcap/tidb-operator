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
if [[ -n "${AWS_DEFAULT_REGION}"]]; then
	EXEC_COMMAND="exec"
else
	EXEC_COMMAND="/usr/local/bin/shush exec --"
fi

# exec command
case "$1" in
    backup)
        shift 1
        echo "$BACKUP_BIN backup $@"
        $EXEC_COMMAND $BACKUP_BIN backup "$@"
        ;;
    export)
        shift 1
        echo "$BACKUP_BIN export $@"
        $EXEC_COMMAND $BACKUP_BIN export "$@"
        ;;
    restore)
        shift 1
        echo "$BACKUP_BIN restore $@"
        $EXEC_COMMAND $BACKUP_BIN restore "$@"
        ;;
    import)
        shift 1
        echo "$BACKUP_BIN import $@"
        $EXEC_COMMAND $BACKUP_BIN import "$@"
        ;;
    clean)
        shift 1
        echo "$BACKUP_BIN clean $@"
        $EXEC_COMMAND $BACKUP_BIN clean "$@"
        ;;
    *)
        echo "Usage: $0 {backup|restore|clean}"
        echo "Now runs your command."
        echo "$@"

        exec "$@"
esac
