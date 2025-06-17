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

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestTiDB(t *testing.T) {
	cases := []Case{}
	cases = append(cases, transferTiDBCases(t, Topology(), "spec", "topology")...)
	cases = append(cases, transferTiDBCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferTiDBCases(t, ServerLabels(), "spec", "server", "labels")...)
	cases = append(cases, transferTiDBCases(t, PodOverlayLabels(), "spec", "overlay", "pod", "metadata")...)
	cases = append(cases, transferTiDBCases(t, mysqlTLS(), "spec", "security", "tls", "mysql")...)
	cases = append(cases, transferTiDBCases(t, OverlayVolumeClaims(), "spec")...)
	cases = append(cases, transferTiDBCases(t, Version(), "spec", "version")...)
	Validate(t, "crd/core.pingcap.com_tidbs.yaml", cases)
}

func basicTiDB() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: TiDB
metadata:
  name: tidb
spec:
  cluster:
    name: test
  subdomain: test
  version: v8.1.0
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}

	return obj
}

func transferTiDBCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicTiDB()
		if c.current == nil {
			unstructured.RemoveNestedField(current, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(current, c.current, fields...))
		}

		c.current = current

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicTiDB()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}

func mysqlTLS() []Case {
	errMsg := `spec.security.tls.mysql: Invalid value: "object": field .mysql.enabled is immutable`
	return []Case{
		{
			desc:     "set enabled to true when creating",
			isCreate: true,
			current:  map[string]any{"enabled": true},
		},
		{
			desc:     "set enabled to false when creating",
			isCreate: true,
			current:  map[string]any{"enabled": false},
		},
		{
			desc:     "try to update enabled from false to true",
			old:      map[string]any{"enabled": false},
			current:  map[string]any{"enabled": true},
			wantErrs: []string{errMsg},
		},
		{
			desc:     "try to update enabled from true to false",
			old:      map[string]any{"enabled": true},
			current:  map[string]any{"enabled": false},
			wantErrs: []string{errMsg},
		},
		{
			desc:    "enabled is not changed",
			old:     map[string]any{"enabled": true},
			current: map[string]any{"enabled": true},
		},
	}
}
