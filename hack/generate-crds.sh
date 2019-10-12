#!/bin/bash -e

GO_PKG="github.com/pingcap/tidb-operator"
scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
to_crdgen="$scriptdir/../cmd/to-crdgen"
crd_target="$scriptdir/../manifests/crd.yaml"

GO111MODULE=on go get k8s.io/code-generator/cmd/openapi-gen@kubernetes-1.12.5

$GOPATH/bin/openapi-gen --go-header-file=$scriptdir/boilerplate.go.txt \
-i $GO_PKG/pkg/apis/pingcap.com/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1 \
-p apis/pingcap.com/v1alpha1  -O openapi_generated -o $scriptdir/../pkg

go install $to_crdgen
$GOPATH/bin/to-crdgen tidbcluster > $crd_target
$GOPATH/bin/to-crdgen backup >> $crd_target
$GOPATH/bin/to-crdgen restore >> $crd_target
$GOPATH/bin/to-crdgen backupschedule >> ${crd_target}
