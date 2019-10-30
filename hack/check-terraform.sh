#!/usr/bin/env bash

tf_version=0.12.12

echo "Version info: $(terraform version)"
if which terraform >/dev/null; then
    version=$(terraform version|head -1|awk '{print $2}')
    if [ ${version} == v${tf_version} ]; then
	TERRAFORM=terraform
    fi
fi

if [ -z "${TERRAFORM}" ]; then
    echo "Installing Terraform (${tf_version})..."
    GO111MODULE=on go get github.com/hashicorp/terraform@v${tf_version}
    TERRAFORM=$GOPATH/bin/terraform
fi

root_dir=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
terraform_modules=$(find ${root_dir}/deploy -not -path '*/\.*' -type f -name variables.tf | xargs dirname)

for module in $terraform_modules; do
    echo "Checking module ${module}"
    cd ${module}
    if ${TERRAFORM} fmt -check >/dev/null; then
	echo "Initialize module ${module}..."
	${TERRAFORM} init >/dev/null
	if ${TERRAFORM} validate > /dev/null; then
	    continue
	else
	    echo "terraform validate failed for ${module}"
	    exit 1
	fi
    else
	echo "terraform fmt -check failed for ${module}"
	terraform fmt
    fi
done
