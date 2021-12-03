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

package tidbngmonitoring

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/pingcap/tidb-operator/pkg/controller"
)

type NGMonitoringStartScriptModel struct {
	TCName          string // name of tidb cluster
	TCNamespace     string // namespace of tidb cluster's namespace
	TCClusterDomain string // cluster domain of tidb cluster

	TNGMName      string // name of tidb ng monitoring
	TNGMNamespace string // namespace of tidb ng monitoring
}

func (m *NGMonitoringStartScriptModel) FormatClusterDomain() string {
	if len(m.TCClusterDomain) > 0 {
		return "." + m.TCClusterDomain
	}
	return ""
}

func (m *NGMonitoringStartScriptModel) PDAddress() string {
	// don't need add scheme, ng monitoring will use https if cert is configured
	return fmt.Sprintf("%s.%s:2379", controller.PDMemberName(m.TCName), m.TCNamespace)
}

func (m *NGMonitoringStartScriptModel) RenderStartScript() (string, error) {
	return renderTemplateFunc(ngMonitoringStartScriptTpl, m)
}

// TODO: Find a better method to start
var ngMonitoringStartScriptTpl = template.Must(template.New("ng-monitoring-start-script").Parse(`/ng-monitoring-server \
	--pd.endpoints {{ .PDAddress }} \
	--advertise-address ${POD_NAME}.${HEADLESS_SERVICE_NAME}:12020 \
	--config /etc/ng-monitoring/ng-monitoring.toml \
	--storage.path /var/lib/ng-monitoring
`))

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
