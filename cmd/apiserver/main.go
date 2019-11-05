// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	_ "github.com/go-openapi/loads"
	"github.com/pingcap/tidb-operator/pkg/apiserver/cmd"
	"github.com/pingcap/tidb-operator/pkg/version"
	_ "github.com/ugorji/go/codec"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Enable cloud provider auth
	"k8s.io/kube-openapi/pkg/common"
)

var emptyOpenAPIDefinitions = func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{}
}

func main() {

	cmd.StartApiServer(nil, emptyOpenAPIDefinitions, "TiDB ApiServer API", version.Get().GitVersion)
}
