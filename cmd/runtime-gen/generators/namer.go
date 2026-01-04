// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generators

import (
	"strings"

	"k8s.io/gengo/v2/types"
)

const (
	nameTiDB    = "TiDB"
	nameTiProxy = "TiProxy"
	nameTiKV    = "TiKV"
	nameTiFlash = "TiFlash"
)

type NameFunc func(t *types.Type) string

func (f NameFunc) Name(t *types.Type) string {
	return f(t)
}

// GroupToInstanceName returns instance name from group type
// e.g. TiDBGroup -> TiDB
func GroupToInstanceName(t *types.Type) string {
	return strings.TrimSuffix(t.Name.Name, "Group")
}

// GroupToSecurityTypeName returns security type name from group type
// TiDB and TiProxy have their own security type
// Others: Security
// TiDB and TiProxy: TiDBSecurity, TiProxySecurity
func GroupToSecurityTypeName(t *types.Type) string {
	name := strings.TrimSuffix(t.Name.Name, "Group")
	switch name {
	case nameTiDB, nameTiProxy:
		return name + "Security"
	}

	return "Security"
}

// GroupToTLSTypeName returns tls type name from group type
// TiDB and TiProxy have their own tls type
// Others: ComponentTLS
// TiDB and TiProxy: TiDBTLSConfig, TiProxyTLSConfig
func GroupToTLSTypeName(t *types.Type) string {
	name := strings.TrimSuffix(t.Name.Name, "Group")
	switch name {
	case nameTiDB, nameTiProxy:
		return name + "TLSConfig"
	}

	return "ComponentTLSConfig"
}

// GroupToInternalTLSTypeName returns internal tls type name from group type
// TiProxy has it's own tls type
// Others: InternalTLS
// TiProxy: InternalClientTLS
func GroupToInternalTLSTypeName(t *types.Type) string {
	if t.Name.Name == "TiProxyGroup" {
		return "InternalClientTLS"
	}

	return "InternalTLS"
}

func InstanceToServerLabelsField(t *types.Type) string {
	switch t.Name.Name {
	case nameTiDB, nameTiKV, nameTiFlash, nameTiProxy:
		return "in.Spec.Server.Labels"
	}

	return "nil"
}
