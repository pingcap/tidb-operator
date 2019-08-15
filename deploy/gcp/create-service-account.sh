#!/usr/bin/env bash
# Create a service account with permissions needed for the terraform
#
# This script is currently designed to be idempotent and re-runnable, like terraform.
#
# We could write this in terraform, but there is a bootstrapping issue,
# so it cannot just be added to the existing terraform.

set -euo pipefail
cd "$(dirname "$0")"
PROJECT="${TF_VAR_GCP_PROJECT:-$(cat terraform.tfvars | awk -F '=' '/GCP_PROJECT/ {print $2}' | cut -d '"' -f 2)}"
echo "using project: $PROJECT"

cred_file=credentials.auto.tfvars
if test -f "$cred_file" ; then
  if cat "$cred_file" | awk -F'=' '/GCP_CREDENTIALS/ {print $2}' >/dev/null ; then
    echo "GCP_CREDENTAILS_PATH already set in $cred_file"
    exit 1
  fi
fi

GCLOUD="gcloud --project $PROJECT"

mkdir -p credentials
key_file=credentials/terraform-key.json
email="terraform@${PROJECT}.iam.gserviceaccount.com"

sas=$($GCLOUD iam service-accounts list)
if echo "$sas" | grep terraform >/dev/null ; then
  if test -f $key_file && grep "$PROJECT" $key_file >/dev/null ; then
    echo "service account terraform already exists along with the key file. Will set terraform variables"
  else
    echo "service account terraform already exists, will get a key for it"
    $GCLOUD iam service-accounts keys create $key_file --iam-account "$email"
  fi
else
  echo "creating a new service account terraform"
  $GCLOUD iam service-accounts create --display-name terraform terraform
  $GCLOUD iam service-accounts keys create $key_file --iam-account "$email"
fi

chmod 0600 $key_file

$GCLOUD projects add-iam-policy-binding "$PROJECT" --member "serviceAccount:$email" --role roles/container.clusterAdmin
$GCLOUD projects add-iam-policy-binding "$PROJECT" --member "serviceAccount:$email" --role roles/compute.networkAdmin
$GCLOUD projects add-iam-policy-binding "$PROJECT" --member "serviceAccount:$email" --role roles/compute.viewer
$GCLOUD projects add-iam-policy-binding "$PROJECT" --member "serviceAccount:$email" --role roles/compute.securityAdmin
$GCLOUD projects add-iam-policy-binding "$PROJECT" --member "serviceAccount:$email" --role roles/iam.serviceAccountUser
$GCLOUD projects add-iam-policy-binding "$PROJECT" --member "serviceAccount:$email" --role roles/compute.instanceAdmin.v1

echo GCP_CREDENTIALS_PATH="\"$(pwd)/$key_file\"" > "$cred_file"
