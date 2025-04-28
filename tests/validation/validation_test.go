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
)

// Test for
// - +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
// - +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when created"
func TestTopologyImmutable(t *testing.T) {
	cases := []Case{
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

	Validate(t, cases)
}
