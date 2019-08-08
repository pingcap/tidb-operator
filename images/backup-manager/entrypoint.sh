#!/bin/sh
set -e

echo "Create rclone.conf file."
cat <<EOF > /tmp/rclone.conf
[s3]
type = s3
env_auth = false
provider = ${S3_PROVIDER:-"AWS"}
access_key_id = ${AWS_ACCESS_KEY_ID}
secret_access_key = ${AWS_SECRET_ACCESS_KEY:-$AWS_SECRET_KEY}
region = ${AWS_REGION:-"us-east-1"}
endpoint = ${S3_ENDPOINT}
acl = ${AWS_ACL}
storage_class = ${AWS_STORAGE_CLASS}
[ceph]
type = s3
env_auth = false
provider = ${S3_PROVIDER:-"Ceph"}
access_key_id = ${AWS_ACCESS_KEY_ID}
secret_access_key = ${AWS_SECRET_ACCESS_KEY:-$AWS_SECRET_KEY}
region = :default-placement
endpoint = ${S3_ENDPOINT}
[gs]
type = google cloud storage
project_number = ${GCS_PROJECT_ID}
service_account_file = /tmp/google-credentials.json
object_acl = ${GCS_OBJECT_ACL}
bucket_acl = ${GCS_BUCKET_ACL}
location =  ${GCS_LOCATION}
storage_class = ${GCS_STORAGE_CLASS:-"MULTI_REGIONAL"}
[azure]
type = azureblob
account = ${AZUREBLOB_ACCOUNT}
key = ${AZUREBLOB_KEY}
EOF

if [[ -n "${GCS_SERVICE_ACCOUNT_JSON_KEY:-}" ]]; then
    echo "Create google-credentials.json file."
    cat <<EOF > /tmp/google-credentials.json
    ${GCS_SERVICE_ACCOUNT_JSON_KEY}
EOF
else
    touch /tmp/google-credentials.json
fi

BACKUP_BIN=/tidb-backup-manager

# exec command
case "$1" in
    backup)
        shift 1
        $BACKUP_BIN backup "$@"
        ;;
    restore)
        shift 1
        $BACKUP_BIN restore "$@"
        ;;
    clean)
        shift 1
        $BACKUP_BIN clean "$@"
        ;;
    schedule-backup)
        shift 1
        $BACKUP_BIN $VERBOSE schedule-backup "$@"
        ;;
    *)
        echo "Usage: $0 {backup|restore|clean}"
        echo "Now runs your command."
        echo "$@"

        exec "$@"
esac
