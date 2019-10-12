#!/usr/bin/env bash

# crd-groups generates crd or verify crd in CI

if [ "$#" -lt 1 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename $0) <action>

  <action>        the action tu run ( generate, verify )

Examples:
  $(basename $0) generate
EOF
  exit 0
fi

ACTION="$1"
shift 1

GO_PKG="github.com/pingcap/tidb-operator"
CI_GO_PATH="/home/jenkins/workspace/operator_ghpr_e2e_test_kind/go"
scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
to_crdgen="$scriptdir/../cmd/to-crdgen"
crd_target="$scriptdir/../manifests/crd.yaml"
crd_verify_target="$scriptdir/../manifests/crd-verify.yaml"

GO111MODULE=on go get k8s.io/code-generator/cmd/openapi-gen@kubernetes-1.12.5

$GOPATH/bin/openapi-gen --go-header-file=$scriptdir/boilerplate.go.txt \
-i $GO_PKG/pkg/apis/pingcap.com/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1 \
-p apis/pingcap.com/v1alpha1  -O openapi_generated -o $scriptdir/../pkg

go install $to_crdgen

function generate_crd {
	$1/bin/to-crdgen generate tidbcluster > $2
	$1/bin/to-crdgen generate backup >> $2
	$1/bin/to-crdgen generate restore >> $2
	$1/bin/to-crdgen generate backupschedule >> $2
}

if test $ACTION == 'generate' ;then
	generate_crd $GOPATH $crd_target
elif [ $ACTION == 'verify' ];then
	generate_crd $CI_GO_PATH $crd_verify_target
	r="$(diff "$crd_target" "$crd_verify_target")"
	if [[ -n $r ]]; then
		echo $1 is not latest
		exit 1
	fi
	echo crds are latest
fi

cd $scriptdir/..
go mod tidy
