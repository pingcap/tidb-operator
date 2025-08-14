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

type NameFunc func(t *types.Type) string

func (f NameFunc) Name(t *types.Type) string {
	return f(t)
}

func GroupToInstanceName(t *types.Type) string {
	return strings.TrimSuffix(t.Name.Name, "Group")
}

func GroupToSecurityTypeName(t *types.Type) string {
	name := strings.TrimSuffix(t.Name.Name, "Group")
	switch name {
	case "TiDB", "TiProxy":
		return name + "Security"
	}

	return "Security"
}

func GroupToTLSTypeName(t *types.Type) string {
	name := strings.TrimSuffix(t.Name.Name, "Group")
	switch name {
	case "TiDB", "TiProxy":
		return name + "TLS"
	}

	return "ComponentTLS"
}

func GroupToInternalTLSTypeName(t *types.Type) string {
	switch t.Name.Name {
	case "TiProxyGroup":
		return "InternalClientTLS"
	}

	return "InternalTLS"
}
