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
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/tidbcloud/tidb-operator/tests/validation/third_party/apiextensions-apiserver/pkg/test"
)

// Test for
// - +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
// - +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when created"
func TestTopologyImmutable(t *testing.T) {
	cases := []struct {
		desc         string
		current, old any
		wantErrs     []string
	}{
		{
			desc:    "topology is always unset",
			old:     map[string]any{},
			current: map[string]any{},
		},
		{
			desc: "topology is not changed",
			old: map[string]any{
				"topology": map[string]any{
					"aaa": "bbb",
				},
			},
			current: map[string]any{
				"topology": map[string]any{
					"aaa": "bbb",
				},
			},
		},
		{
			desc: "topology is changed",
			old: map[string]any{
				"topology": map[string]any{
					"aaa": "bbb",
				},
			},
			current: map[string]any{
				"topology": map[string]any{
					"aaa": "ccc",
				},
			},
			wantErrs: []string{
				`openAPIV3Schema.properties.spec.topology: Invalid value: "object": topology is immutable`,
			},
		},
		{
			desc: "set topology",
			old:  map[string]any{},
			current: map[string]any{
				"topology": map[string]any{
					"aaa": "ccc",
				},
			},
			wantErrs: []string{
				`openAPIV3Schema.properties.spec.topology: Invalid value: "object": topology can only be set when created`,
			},
		},
		{
			desc: "unset topology",
			old: map[string]any{
				"topology": map[string]any{
					"aaa": "ccc",
				},
			},
			current: map[string]any{},
			wantErrs: []string{
				`openAPIV3Schema.properties.spec.topology: Invalid value: "object": topology can only be set when created`,
			},
		},
	}
	for _, crd := range InstanceCRDs {
		validators := test.FieldValidators(t,
			test.MustLoadManifest[apiextensionsv1.CustomResourceDefinition](t, crd))

		path := "openAPIV3Schema.properties.spec"
		validator, found := validators["v1alpha1"][path]
		require.True(t, found, "failed to find validator for %s", path)

		for i := range cases {
			c := &cases[i]

			t.Run(c.desc, func(tt *testing.T) {
				tt.Parallel()
				errs := validator(c.current, c.old)
				require.Equal(tt, len(c.wantErrs), len(errs), "%s: %v", c.desc, errs)
				for i, err := range errs {
					assert.EqualError(tt, err, c.wantErrs[i], c.desc)
				}
			})
		}

	}
}
