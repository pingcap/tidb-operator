#!/usr/bin/env bash
# Create a service account with permissions needed for the terraform
#
# This script is currently designed to be idempotent and re-runnable, like terraform.
#
# We could write this in terraform, but there is a bootstrapping issue,
# so it cannot just be added to the existing terraform.

set -euo pipefail

if ! cd "$(dirname "$0")"; then
    printf "Could not change to base directory of script." >&2
    exit 1
fi

if ! [[ -d .terraform ]]; then
    echo "no .terraform directory, perhaps you need to run ''terraform init''?" >&2
    exit 1
fi

if ! project=${TF_VAR_GCP_PROJECT:-$( echo var.GCP_PROJECT | terraform console 2>/dev/null )} || [[ -z $project ]]; then
    echo "could not identify current project; set GCP_PROJECT in a .tfvars file or set the TF_VAR_GCP_PROJECT environment variable" >&2
    exit 1
fi
echo "using project: $project"

cred_file=credentials.auto.tfvars

if cred_path=$( echo var.GCP_CREDENTIALS_PATH | terraform console 2>/dev/null ) && [[ $cred_path ]]; then
    if ! command -v jq >/dev/null; then
        echo "GCP_CREDENTIALS_PATH already set to $cred_path and jq(1) is not installed to ensure it is for project $project" >&2
        exit 1
    elif cred_project=$(jq -r .project_id "$cred_path" 2>/dev/null) && [[ $cred_project != "$project" ]]; then
        echo "GCP_CREDENTIALS_PATH already set to $cred_path but credentials project $cred_project does not match current project $project" >&2
        exit 1
    elif ! [[ -f $cred_path ]]; then
        echo "GCP_CREDENTIALS_PATH already set, but $cred_path doesn't exist" >&2
    else
        echo "GCP_CREDENTIALS_PATH already set to $cred_path for project $project" >&2
        exit
    fi
fi

gcloud=( gcloud --project "$project" )

mkdir -p credentials
key_file=credentials/terraform-key.json
email="terraform@${project}.iam.gserviceaccount.com"

if [[ $("${gcloud[@]}" --format='value(name)' iam service-accounts list --filter='displayName~^terraform$') ]]; then
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
echo "Successfully wrote credentials to $key_file and configuration to $cred_file"
