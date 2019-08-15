#!/usr/bin/env bash
# Create a service account with permissions needed for the terraform
#
# This script is currently designed to be idempotent and re-runnable, like terraform.
#
# We could write this in terraform, but there is a bootstrapping issue,
# so it cannot just be added to the existing terraform.

set -euo pipefail
set -x
if ! cd "$(dirname "$0")"; then
    printf "Could not change to base directory of script." >&2
    exit 1
fi

project=${TF_VAR_GCP_PROJECT:-$( echo var.GCP_PROJECT | terraform console )}
echo "using project: $project"

cred_file=credentials.auto.tfvars

#cred_path=$( terraform console <<<var.GCP_CREDENTIALS_PATH 2>/dev/null )
if cred_path=$( echo var.GCP_CREDENTIALS_PATH | terraform console 2>/dev/null ) && [[ $cred_path ]]; then
    echo "GCP_CREDENTAILS_PATH already set to $cred_path"
    exit 1
fi

gcloud=( gcloud --project "$project" )

mkdir -p credentials
key_file=credentials/terraform-key.json
email="terraform@${project}.iam.gserviceaccount.com"

if [[ $("${gcloud[@]}" --format='value(name)' iam service-accounts list --filter=displayName:terraform) ]]; then
  if grep -sq "$project" "$key_file"; then
    echo "service account terraform already exists along with the key file, will set terraform variables"
  else
    echo "service account terraform already exists, will get a key for it"
    "${gcloud[@]}" iam service-accounts keys create "$key_file" --iam-account "$email"
  fi
else
  echo "creating a new service account terraform"
  "${gcloud[@]}" iam service-accounts create --display-name terraform terraform
  "${gcloud[@]}" iam service-accounts keys create "$key_file" --iam-account "$email"
fi

chmod 0600 "$key_file"

roles=(
  roles/container.clusterAdmin
  roles/compute.networkAdmin
  roles/compute.viewer
  roles/compute.securityAdmin
  roles/iam.serviceAccountUser
  roles/compute.instanceAdmin.v1
)
for role in "${roles[@]}"; do 
    "${gcloud[@]}" projects add-iam-policy-binding "$project" --member "serviceAccount:$email" --role "$role"
done

printf 'GCP_CREDENTIALS_PATH = "%s/%s"\n' "$PWD" "$key_file" > "$cred_file"
