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
	"fmt"
	"text/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

// TiFlashStartScriptModel contain fields for rendering TiFlash start script
type TiFlashStartScriptModel struct {
	ExtraArgs string
}

// RenderTiFlashStartScript renders TiFlash start script from TidbCluster
func RenderTiFlashStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	if tc.Spec.TiFlash.IsInitScriptDisabled() {
		return RenderTiFlashStartScriptWithStartArgs(tc)
	}

	m := &TiFlashStartScriptModel{}

	m.ExtraArgs = ""

	return renderTemplateFunc(tiflashStartScriptTpl, m)
}

// TiFlashStartScriptWithStartArgsModel is an alternative of TiFlashStartScriptModel that enable tiflash pod run without
// initContainer
type TiFlashStartScriptWithStartArgsModel struct {
	AdvertiseAddr             string
	EnableAdvertiseStatusAddr bool
	AdvertiseStatusAddr       string
	Addr                      string
	ExtraArgs                 string
}

func RenderTiFlashStartScriptWithStartArgs(tc *v1alpha1.TidbCluster) (string, error) {
	model := &TiFlashStartScriptWithStartArgsModel{
		AdvertiseAddr:             fmt.Sprintf("${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:%d", controller.FormatClusterDomain(tc.Spec.ClusterDomain), v1alpha1.DefaultTiFlashProxyPort),
		EnableAdvertiseStatusAddr: false,
		ExtraArgs:                 "",
	}
	// only the tiflash learner supports dynamic configuration
	if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
		model.AdvertiseStatusAddr = fmt.Sprintf("${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:%d", controller.FormatClusterDomain(tc.Spec.ClusterDomain), v1alpha1.DefaultTiFlashProxyStatusPort)
		model.EnableAdvertiseStatusAddr = true
	}

	listenHost := "0.0.0.0"
	if tc.Spec.PreferIPv6 {
		listenHost = "[::]"
	}
	model.Addr = fmt.Sprintf("%s:%d", listenHost, v1alpha1.DefaultTiFlashFlashPort)

	return renderTemplateFunc(tiflashStartScriptWithStartArgsTpl, model)
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
	// tiflashStartScriptWithStartArgs is the new way to launch tiflash, that enable tiflash pod run without init container.
	tiflashStartScriptWithStartArgs = `
# Use HOSTNAME if POD_NAME is unset for backward compatibility.
POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="server --config-file /etc/tiflash/config_templ.toml \
-- \
--flash.proxy.advertise-addr={{ .AdvertiseAddr }} \{{if .EnableAdvertiseStatusAddr }}
--flash.proxy.advertise-status-addr={{ .AdvertiseStatusAddr }} \{{end}}
--flash.service_addr={{ .Addr }}
"

{{- if .ExtraArgs }}
ARGS="${ARGS} {{ .ExtraArgs }}"
{{- end }}

if [ ! -z "${STORE_LABELS:-}" ]; then
  LABELS=" --labels ${STORE_LABELS} "
  ARGS="${ARGS}${LABELS}"
fi

echo "starting tiflash-server ..."
echo "/tiflash/tiflash ${ARGS}"
exec /tiflash/tiflash ${ARGS}
`
)

var tiflashStartScriptTpl = template.Must(
	template.Must(
		template.New("tiflash-start-script").Parse(tiflashStartSubScript),
	).Parse(componentCommonScript + tiflashStartScript),
)

var tiflashStartScriptWithStartArgsTpl = template.Must(
	template.Must(
		template.New("tiflash-start-script-with-start-args").Parse(tiflashStartSubScript),
	).Parse(componentCommonScript + tiflashStartScriptWithStartArgs),
)
