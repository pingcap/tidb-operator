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
	"bytes"
	"text/template"
)

const (
	componentCommonScript = `#!/bin/sh

set -uo pipefail

ANNOTATIONS="/etc/podinfo/annotations"
OPERATOR_ENV="/etc/operator.env"

if [[ ! -f "${ANNOTATIONS}" ]]
then
    echo "${ANNOTATIONS} does't exist, exiting."
    exit 1
fi
source ${ANNOTATIONS} 2>/dev/null

runmode=${runmode:-normal}

if [[ X${runmode} == Xdebug ]]
then
    echo "entering debug mode."
    tail -f /dev/null
fi
`
)

type CommonModel struct {
	ClusterName      string // same as tc.metadata.name
	ClusterNamespace string // same as tc.metadata.namespace
	ClusterDomain    string // same as tc.spec.clusterDomain
	PeerServiceName  string // the name of the peer service
	AcrossK8s        bool   // whether the cluster is deployed across k8s
}

func (m CommonModel) FormatClusterDomain() string {
	if len(m.ClusterDomain) > 0 {
		return "." + m.ClusterDomain
	}
	return ""
}

func renderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}
