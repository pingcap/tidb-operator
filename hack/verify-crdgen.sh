#!/usr/bin/env bash

GO_PKG="github.com/pingcap/tidb-operator"
scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
to_crdgen="$scriptdir/../cmd/to-crdgen/main.go"
crd_target="$scriptdir/../manifests/crd.yaml"
crd_verify_target="$scriptdir/../manifests/crd-verify.yaml"

GO111MODULE=on go get k8s.io/code-generator/cmd/openapi-gen@kubernetes-1.12.5

$GOPATH/bin/openapi-gen --go-header-file=$scriptdir/boilerplate.go.txt \
-i $GO_PKG/pkg/apis/pingcap.com/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1 \
-p apis/pingcap.com/v1alpha1  -O openapi_generated -o ../pkg

go run $to_crdgen tidbcluster > $crd_verify_target
go run $to_crdgen backup >> $crd_verify_target
go run $to_crdgen restore >> $crd_verify_target
go run $to_crdgen backupschedule >> $crd_verify_target

r="$(diff "$crd_target" "$crd_verify_target")"
if [[ -n $r ]]; then
	echo $1 is not latest
	exit 1
fi

echo crds are latest
