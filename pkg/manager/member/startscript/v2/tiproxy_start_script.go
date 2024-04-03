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

// TiProxyStartScriptModel contain fields for rendering TiProxy start script
type TiProxyStartScriptModel struct {
	AdvertiseAddr string
}

// RenderTiProxyStartScript renders tiproxy start script for TidbCluster
func RenderTiProxyStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	m := &TiProxyStartScriptModel{}
	tcName := tc.Name
	tcNS := tc.Namespace
	peerServiceName := controller.TiProxyPeerMemberName(tcName)
	m.AdvertiseAddr = fmt.Sprintf("${TIPROXY_POD_NAME}.%s.%s.svc", peerServiceName, tcNS)
	if tc.Spec.ClusterDomain != "" {
		m.AdvertiseAddr = m.AdvertiseAddr + "." + tc.Spec.ClusterDomain
	}
	return renderTemplateFunc(template.Must(template.New("tiproxy").Parse(componentCommonScript+tiproxyStartScript)), m)
}

// Old TiProxy versions don't support advertise-addr.
// We can't add advertise-addr to the config file because the file is read-only.
const (
	tiproxyStartScript = `
TIPROXY_POD_NAME=${POD_NAME:-$HOSTNAME}
ARGS="--config=/etc/proxy/proxy.toml"
if [[ "$(/bin/tiproxy --help)" == *"advertise-addr"* ]]; then
  ARGS="${ARGS} --advertise-addr={{ .AdvertiseAddr }}"
fi

echo "starting: tiproxy ${ARGS}"
exec /bin/tiproxy ${ARGS}
`
)
