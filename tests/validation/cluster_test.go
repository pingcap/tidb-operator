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

func TestCluster(t *testing.T) {
	var cases []Case
	cases = append(cases, transferClusterCases(t, FeatureGates(), "spec", "featureGates")...)
	cases = append(cases, transferClusterCases(t, bootstrapSQL(), "spec", "bootstrapSQL")...)
	cases = append(cases, transferClusterCases(t, tlsCluster(), "spec", "tlsCluster")...)

	Validate(t, "crd/core.pingcap.com_clusters.yaml", cases)
}

func basicCluster() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: Cluster
metadata:
  name: basic
spec: {}
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}

	return obj
}

func transferClusterCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicCluster()
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

		old := basicCluster()
		if c.old == nil {
			unstructured.RemoveNestedField(old, fields...)
		} else {
			require.NoError(t, unstructured.SetNestedField(old, c.old, fields...))
		}

		c.old = old
	}

	return cases
}

func tlsCluster() []Case {
	errMsg := `spec.tlsCluster: Invalid value: "object": field .tlsCluster.enabled is immutable`
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

func bootstrapSQL() []Case {
	errMsg := `spec: Invalid value: "object": bootstrapSQL can only be set at creation, can be unset, but cannot be changed to a different value`
	return []Case{
		{
			desc:    "no change",
			old:     map[string]any{"name": "bootstrap-sql"},
			current: map[string]any{"name": "bootstrap-sql"},
		},
		{
			desc:     "set bootstrapSQL",
			isCreate: true,
			current:  map[string]any{"name": "bootstrap-sql"},
		},
		{
			desc:     "unset bootstrapSQL",
			isCreate: true,
			current:  nil,
		},
		{
			desc: "bootstrapSQL is always unset",
		},
		{
			desc:     "try to change bootstrapSQL",
			old:      map[string]any{"name": "bootstrap-sql-old"},
			current:  map[string]any{"name": "bootstrap-sql-new"},
			wantErrs: []string{errMsg},
		},
		{
			desc:    "try to set bootstrapSQL to nil",
			old:     map[string]any{"name": "bootstrap-sql-old"},
			current: nil,
		},
		{
			desc:    "update with both nil",
			old:     nil,
			current: nil,
		},
		{
			desc:     "update from nil to non-nil",
			old:      nil,
			current:  map[string]any{"name": "bootstrap-sql"},
			wantErrs: []string{errMsg},
		},
		{
			desc:    "update from non-nil to nil",
			old:     map[string]any{"name": "bootstrap-sql"},
			current: nil,
		},
	}
}
