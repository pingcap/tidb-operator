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

	"github.com/stretchr/testify/assert"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
)

func TestClusterFeatureGateEnabledDefault(t *testing.T) {
	schema := structuralSchemaFromCRD(t, "crd/core.pingcap.com_clusters.yaml", "v1alpha1")
	for _, tc := range []struct {
		name string
		gate map[string]any
		want bool
	}{
		{"omitted", map[string]any{"name": "VolumeAttributesClass"}, true},
		{"false", map[string]any{"name": "VolumeAttributesClass", "enabled": false}, false},
		{"true", map[string]any{"name": "VolumeAttributesClass", "enabled": true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := map[string]any{"spec": map[string]any{"featureGates": []any{tc.gate}}}
			defaulting.Default(obj, schema)
			assert.Equal(t, tc.want, tc.gate["enabled"])
		})
	}
}

func TestClusterExplicitFeatureModification(t *testing.T) {
	modification := func(enabled bool) map[string]any {
		return map[string]any{"name": "FeatureModification", "enabled": enabled}
	}
	other := func(enabled bool) map[string]any {
		return map[string]any{"name": "VolumeAttributesClass", "enabled": enabled}
	}
	cases := []Case{
		{
			desc: "explicitly disable an ordinary feature when modification is enabled",
			old:  []any{modification(true), other(true)}, current: []any{modification(true), other(false)},
		},
		{
			desc: "explicitly false FeatureModification does not unlock changes",
			old:  []any{modification(false), other(true)}, current: []any{modification(false), other(false)},
			wantErrs: []string{`spec.featureGates: Invalid value: "array": can only enable FeatureModification if it's not enabled`},
		},
		{
			desc: "cannot explicitly turn off FeatureModification",
			old:  []any{modification(true)}, current: []any{modification(false)},
			wantErrs: []string{`spec.featureGates: Invalid value: "array": cannot disable FeatureModification`},
		},
		{
			desc: "enable previously explicitly disabled FeatureModification",
			old:  []any{modification(false), other(true)}, current: []any{modification(true), other(true)},
		},
	}
	Validate(t, "crd/core.pingcap.com_clusters.yaml", transferClusterCases(t, cases, "spec", "featureGates"))
}
