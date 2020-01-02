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
CI_GO_PATH="/go"
scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
to_crdgen="$scriptdir/../cmd/to-crdgen"
crd_target="$scriptdir/../manifests/crd.yaml"
crd_verify_tmp=$(mktemp)
trap "rm $crd_verify_tmp" EXIT

export GO111MODULE=on

go install k8s.io/code-generator/cmd/openapi-gen

function generate_crd {
    $1/bin/openapi-gen --go-header-file=$scriptdir/boilerplate/boilerplate.generatego.txt \
    -i $GO_PKG/pkg/apis/pingcap/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1 \
    -p apis/pingcap/v1alpha1  -O openapi_generated -o $scriptdir/../pkg

    go install $to_crdgen

	$1/bin/to-crdgen generate tidbcluster > $2
	$1/bin/to-crdgen generate backup >> $2
	$1/bin/to-crdgen generate restore >> $2
	$1/bin/to-crdgen generate backupschedule >> $2
	$1/bin/to-crdgen generate tidbmonitor >> $2
	$1/bin/to-crdgen generate tidbinitializer >> $2
}

if test $ACTION == 'generate' ;then
	generate_crd $GOPATH $crd_target
elif [ $ACTION == 'verify' ];then

    if [[ $GOPATH == /go* ]] ;
    then
        generate_crd $CI_GO_PATH $crd_verify_tmp
    else
        generate_crd $GOPATH $crd_verify_tmp
    fi

    echo "diffing $crd_target with $crd_verify_tmp" >&2
	r="$(diff "$crd_target" "$crd_verify_tmp")"
	if [[ -n $r ]]; then
		echo $crd_target is not latest
		exit 1
	fi
	echo crds are latest
fi
