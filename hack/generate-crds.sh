#!/bin/bash -e

# Copyright 2017 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

GO_PKG="github.com/pingcap/tidb-operator"
scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
to_crdgen="${scriptdir}/../cmd/to-crdgen/main.go"
crd_target="${scriptdir}/../manifests/crd.yaml"

GO111MODULE=on go get k8s.io/code-generator/cmd/openapi-gen@kubernetes-1.12.5

${GOPATH}/bin/openapi-gen --go-header-file=${scriptdir}/boilerplate.go.txt \
    -i ${GO_PKG}/pkg/apis/pingcap.com/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1 \
    -p apis/pingcap.com/v1alpha1  -O openapi_generated -o ../pkg

go run ${to_crdgen} tidbcluster > ${crd_target}
go run ${to_crdgen} backup >> ${crd_target}
go run ${to_crdgen} restore >> ${crd_target}
go run ${to_crdgen} backupschedule >> ${crd_target}
