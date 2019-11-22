#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
source $ROOT/hack/lib.sh

hack::ensure_terraform

terraform_modules=$(find ${ROOT}/deploy -not -path '*/\.*' -type f -name variables.tf | xargs -I{} -n1 dirname {})

for module in $terraform_modules; do
    echo "Checking module ${module}"
    cd ${module}
    if ${TERRAFORM_BIN} fmt -check >/dev/null; then
	echo "Initialize module ${module}..."
	${TERRAFORM_BIN} init >/dev/null
	if ${TERRAFORM_BIN} validate > /dev/null; then
	    continue
	else
	    echo "terraform validate failed for ${module}"
	    exit 1
	fi
    else
	echo "terraform fmt -check failed for ${module}"
	${TERRAFORM_BIN} fmt
    fi
done
