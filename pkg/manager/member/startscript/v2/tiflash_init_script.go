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

// TiFlashInitScriptModel contain fields for rendering TiFlash Init script
type TiFlashInitScriptModel struct {
	AcrossK8s *AcrossK8sScriptModel
}

// RenderTiFlashInitScript renders TiFlash Init script from TidbCluster
func RenderTiFlashInitScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiFlashInitScriptModel{}
	tcName := tc.Name
	tcNS := tc.Namespace

	if tc.AcrossK8s() {
		m.AcrossK8s = &AcrossK8sScriptModel{
			PDAddr:        fmt.Sprintf("%s://%s:2379", tc.Scheme(), controller.PDMemberName(tcName)),
			DiscoveryAddr: fmt.Sprintf("%s-discovery.%s:10261", tcName, tcNS),
		}
	}

	return renderTemplateFunc(tiflashInitScriptTpl, m)
}

const (
	// tiflashInitSubScript contains optional subscripts used in start script.
	tiflashInitSubScript = `
{{ define "AcrossK8sSubscript" }}
pd_url={{ .AcrossK8s.PDAddr }}
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url={{ .AcrossK8s.DiscoveryAddr }}
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null | sed 's/http:\/\///g' | sed 's/https:\/\///g'); do
    echo "waiting for the verification of PD endpoints ..."
    sleep 2
done

sed -i s/PD_ADDR/${result}/g /data0/config.toml
sed -i s/PD_ADDR/${result}/g /data0/proxy.toml
{{- end }}
`

	// tiflashInitScript is the template of start script.
	tiflashInitScript = `#!/bin/sh

set -uo pipefail

ordinal=$(echo ${POD_NAME} | awk -F- '{print $NF}')
sed s/POD_NUM/${ordinal}/g /etc/tiflash/config_templ.toml > /data0/config.toml
sed s/POD_NUM/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data0/proxy.toml

{{- if .AcrossK8s -}} {{ template "AcrossK8sSubscript" . }} {{- end }}
`
)

var tiflashInitScriptTpl = template.Must(
	template.Must(
		template.New("tiflash-init-script").Parse(tiflashInitSubScript),
	).Parse(tiflashInitScript),
)
