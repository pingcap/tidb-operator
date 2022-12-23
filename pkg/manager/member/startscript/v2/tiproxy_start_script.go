// Copyright 2021 PingCAP, Inc.
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

package v2

import (
	"text/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

// TiProxyStartScriptModel contain fields for rendering TiProxy start script
type TiProxyStartScriptModel struct {
}

// RenderTiProxyStartScript renders tiproxy start script for TidbCluster
func RenderTiProxyStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiProxyStartScriptModel{}
	return renderTemplateFunc(template.Must(template.New("tiproxy").Parse(componentCommonScript+`
ARGS="--config=/etc/proxy/proxy.toml"
echo "starting: tiproxy ${ARGS}"
exec /bin/tiproxy ${ARGS}
`)), m)
}
