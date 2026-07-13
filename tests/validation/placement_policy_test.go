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

	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestPlacementPolicy(t *testing.T) {
	var cases []Case
	cases = append(cases, transferPlacementPolicyCases(t, ClusterReference(), "spec", "cluster")...)
	cases = append(cases, transferPlacementPolicyCases(t, PlacementPolicyGroupRefs(), "spec", "groupRefs")...)
	cases = append(cases, transferPlacementPolicyCases(t, PlacementPolicyRules(), "spec", "rules")...)
	Validate(t, "crd/core.pingcap.com_placementpolicies.yaml", cases)
}

func PlacementPolicyGroupRefs() []Case {
	return []Case{
		{
			desc:     "groupRefs cannot be empty",
			isCreate: true,
			current:  []any{},
			wantErrs: []string{
				`spec.groupRefs: Invalid value: 0: spec.groupRefs in body should have at least 1 items`,
			},
		},
		{
			desc:     "groupRef group is restricted",
			isCreate: true,
			current: []any{
				map[string]any{
					"group": "other.pingcap.com",
					"kind":  "TiKVGroup",
					"name":  "g1",
				},
			},
			wantErrs: []string{
				`spec.groupRefs[0].group: Unsupported value: "other.pingcap.com": supported values: "core.pingcap.com"`,
			},
		},
		{
			desc:     "groupRef kind is restricted",
			isCreate: true,
			current: []any{
				map[string]any{
					"group": "core.pingcap.com",
					"kind":  "PDGroup",
					"name":  "g1",
				},
			},
			wantErrs: []string{
				`spec.groupRefs[0].kind: Unsupported value: "PDGroup": supported values: "TiKVGroup"`,
			},
		},
	}
}

func PlacementPolicyRules() []Case {
	return []Case{
		{
			desc:     "rules cannot be empty",
			isCreate: true,
			current:  []any{},
			wantErrs: []string{
				`spec.rules: Invalid value: 0: spec.rules in body should have at least 1 items`,
			},
		},
		{
			desc:     "rule role is restricted",
			isCreate: true,
			current: []any{
				map[string]any{
					"name":  "r1",
					"role":  "leader",
					"count": int64(3),
					"selector": map[string]any{
						"keyspace": map[string]any{
							"ids": []any{"1"},
						},
					},
				},
			},
			wantErrs: []string{
				`spec.rules[0].role: Unsupported value: "leader": supported values: "voter"`,
			},
		},
		{
			desc:     "rule count must be positive",
			isCreate: true,
			current: []any{
				map[string]any{
					"name":  "r1",
					"role":  "voter",
					"count": int64(0),
					"selector": map[string]any{
						"keyspace": map[string]any{
							"ids": []any{"1"},
						},
					},
				},
			},
			wantErrs: []string{
				`spec.rules[0].count: Invalid value: 0: spec.rules[0].count in body should be greater than or equal to 1`,
			},
		},
		{
			desc:     "selector keyspace is required",
			isCreate: true,
			current: []any{
				map[string]any{
					"name":     "r1",
					"role":     "voter",
					"count":    int64(3),
					"selector": map[string]any{},
				},
			},
			wantErrs: []string{
				`spec.rules[0].selector: Invalid value: "object": selector.keyspace is required`,
			},
		},
	}
}

func basicPlacementPolicy() map[string]any {
	data := []byte(`
apiVersion: core.pingcap.com/v1alpha1
kind: PlacementPolicy
metadata:
  name: placement-policy
spec:
  cluster:
    name: test
  groupRefs:
  - group: core.pingcap.com
    kind: TiKVGroup
    name: kvg
  rules:
  - name: r1
    role: voter
    count: 3
    selector:
      keyspace:
        ids:
        - "1"
`)
	obj := map[string]any{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		panic(err)
	}

	return obj
}

func transferPlacementPolicyCases(t *testing.T, cases []Case, fields ...string) []Case {
	for i := range cases {
		c := &cases[i]

		current := basicPlacementPolicy()
		c.current = Patch(t, c.mode, current, c.current, fields...)

		if c.isCreate {
			c.old = nil
			continue
		}

		old := basicPlacementPolicy()
		c.old = Patch(t, c.mode, old, c.old, fields...)
	}

	return cases
}
