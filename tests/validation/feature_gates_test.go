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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdvalidation "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	"sigs.k8s.io/yaml"
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

// Rule evaluation alone does not cover the static CEL cost check performed when
// the API server registers a CRD. Exceeding that budget prevents operator startup.
func TestClusterCRDRegistration(t *testing.T) {
	data, err := os.ReadFile("crd/core.pingcap.com_clusters.yaml")
	require.NoError(t, err)
	var crd apiextensionsv1.CustomResourceDefinition
	require.NoError(t, yaml.Unmarshal(data, &crd))
	apiextensionsv1.SetDefaults_CustomResourceDefinition(&crd)
	var internal apiextensions.CustomResourceDefinition
	require.NoError(t, apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(&crd, &internal, nil))
	internal.Status.StoredVersions = []string{"v1alpha1"}
	assert.Empty(t, crdvalidation.ValidateCustomResourceDefinition(t.Context(), &internal))
}
