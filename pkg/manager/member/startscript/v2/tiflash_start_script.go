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

// TiFlashStartScriptModel contain fields for rendering TiFlash start script
type TiFlashStartScriptModel struct {
	ExtraArgs string
}

// RenderTiFlashStartScript renders TiFlash start script from TidbCluster
func RenderTiFlashStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiFlashStartScriptModel{}

	m.ExtraArgs = ""

	return renderTemplateFunc(tiflashStartScriptTpl, m)
}

const (
	// tiflashStartSubScript contains optional subscripts used in start script.
	tiflashStartSubScript = ``

	// tiflashStartScript is the template of start script.
	//
	// Because init container of tiflash have core start script, so just to start tiflash there.
	tiflashStartScript = `
ARGS="--config-file /data0/config.toml"
{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash server ${ARGS}
`
)

var tiflashStartScriptTpl = template.Must(
	template.Must(
		template.New("tiflash-start-script").Parse(tiflashStartSubScript),
	).Parse(componentCommonScript + tiflashStartScript),
)
