#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")"
PROJECT="${TF_VAR_GCP_PROJECT:-$(cat terraform.tfvars | awk -F '=' '/GCP_PROJECT/ {print $2}' | cut -d '"' -f 2)}"
echo "$PROJECT"

cred_file=credentials.auto.tfvars
if test -f "$cred_file" ; then
  if cat "$cred_file" | awk -F'=' '/GCP_CREDENTIALS/ {print $2}' >/dev/null ; then
    echo "GCP_CREDENTAILS_PATH already set in $cred_file"
    exit 1
  fi
fi

gcloud iam service-accounts create --display-name terraform terraform
email="terraform@${PROJECT}.iam.gserviceaccount.com"
gcloud projects add-iam-policy-binding "$PROJECT" --member "$email" --role roles/container.clusterAdmin
gcloud projects add-iam-policy-binding "$PROJECT" --member "$email" --role roles/compute.networkAdmin
gcloud projects add-iam-policy-binding "$PROJECT" --member "$email" --role roles/compute.viewer
gcloud projects add-iam-policy-binding "$PROJECT" --member "$email" --role roles/compute.securityAdmin
gcloud projects add-iam-policy-binding "$PROJECT" --member "$email" --role roles/iam.serviceAccountUser
gcloud projects add-iam-policy-binding "$PROJECT" --member "$email" --role roles/compute.instanceAdmin.v1

mkdir -p credentials
gcloud iam service-accounts keys create credentials/terraform-key.json --iam-account "$email"
echo GCP_CREDENTIALS_PATH="$(pwd)/credentials/terraform-key.json" > "$cred_file"
