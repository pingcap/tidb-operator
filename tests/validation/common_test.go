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

import "fmt"

func NameIsDNSSubdoamin(failMsgFmt string) []Case {
	var cases []Case
	goodValues := []string{
		"a", "ab", "abc", "a1", "a-1", "a--1--2--b",
		"0", "01", "012", "1a", "1-a", "1--a--b--2",
		"a.a", "ab.a", "abc.a", "a1.a", "a-1.a", "a--1--2--b.a",
		"a.1", "ab.1", "abc.1", "a1.1", "a-1.1", "a--1--2--b.1",
		"0.a", "01.a", "012.a", "1a.a", "1-a.a", "1--a--b--2",
		"0.1", "01.1", "012.1", "1a.1", "1-a.1", "1--a--b--2.1",
		"a.b.c.d.e", "aa.bb.cc.dd.ee", "1.2.3.4.5", "11.22.33.44.55",
		// strings.Repeat("a", 253),
	}
	badValues := []string{
		"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
		"-", "a-", "-a", "1-", "-1",
		"_", "a_", "_a", "a_b", "1_", "_1", "1_2",
		".", "a.", ".a", "a..b", "1.", ".1", "1..2",
		" ", "a ", " a", "a b", "1 ", " 1", "1 2",
		"A.a", "aB.a", "ab.A", "A1.a", "a1.A",
		"A.1", "aB.1", "A1.1", "1A.1",
		"0.A", "01.A", "012.A", "1A.a", "1a.A",
		"A.B.C.D.E", "AA.BB.CC.DD.EE", "a.B.c.d.e", "aa.bB.cc.dd.ee",
		"a@b", "a,b", "a_b", "a;b",
		"a:b", "a%b", "a?b", "a$b",
		// strings.Repeat("a", 254),
	}
	for _, val := range goodValues {
		cases = append(cases, Case{
			desc:     fmt.Sprintf("name '%s' is dns 1123 subdomain", val),
			isCreate: true,
			current: map[string]any{
				"name": val,
			},
		})
	}
	for _, val := range badValues {
		cases = append(cases, Case{
			desc:     fmt.Sprintf("name '%s' is not dns 1123 subdomain", val),
			isCreate: true,
			current: map[string]any{
				"name": val,
			},
			wantErrs: []string{
				fmt.Sprintf(failMsgFmt, val),
			},
		})
	}

	return cases
}

func ClusterReference() []Case {
	nameFailMsgFmt := `spec.cluster.name: Invalid value: "%s": spec.cluster.name in body should match '^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$'`
	cases := []Case{
		{
			desc:     "cluster ref cannot be empty",
			isCreate: true,
			current:  map[string]any{},
			wantErrs: []string{
				`spec.cluster.name: Required value`,
			},
		},
		{
			desc:     "cluster name cannot be empty",
			isCreate: true,
			current: map[string]any{
				"name": "",
			},
			wantErrs: []string{
				fmt.Sprintf(nameFailMsgFmt, ""),
			},
		},
		{
			desc: "cluster name cannot be changed",
			current: map[string]any{
				"name": "bbb",
			},
			old: map[string]any{
				"name": "aaa",
			},
			wantErrs: []string{
				`spec.cluster.name: Invalid value: "string": cluster name is immutable`,
			},
		},
	}

	cases = append(cases, NameIsDNSSubdoamin(nameFailMsgFmt)...)

	return cases
}

func Topology() []Case {
	cases := []Case{
		{
			desc:     "topology is supported",
			isCreate: true,
			current: map[string]any{
				"aaa": "bbb",
			},
		},
		{
			desc:     "empty topology is not supported",
			isCreate: true,
			current:  map[string]any{},
			wantErrs: []string{
				`spec.topology: Invalid value: 0: spec.topology in body should have at least 1 properties`,
			},
		},
		{
			desc: "topology is always unset",
		},
		{
			desc: "topology is not changed",
			old: map[string]any{
				"aaa": "bbb",
			},
			current: map[string]any{
				"aaa": "bbb",
			},
		},
		{
			desc: "topology is changed",
			old: map[string]any{
				"aaa": "bbb",
			},
			current: map[string]any{
				"aaa": "ccc",
			},
			wantErrs: []string{
				`spec.topology: Invalid value: "object": topology is immutable`,
			},
		},
		{
			desc: "set topology",
			current: map[string]any{
				"aaa": "ccc",
			},
			wantErrs: []string{
				`spec.topology: Invalid value: "object": topology can only be set when creating`,
			},
		},
		{
			desc: "unset topology",
			old: map[string]any{
				"aaa": "ccc",
			},
			wantErrs: []string{
				`spec.topology: Invalid value: "object": topology can only be set when creating`,
			},
		},
	}

	return cases
}

func PodOverlayLabels() []Case {
	cases := []Case{
		{
			desc:     "no label overlay is ok",
			isCreate: true,
			current:  map[string]any{},
		},
		{
			desc:     "ok to overlay normal labels",
			isCreate: true,
			current: map[string]any{
				"labels": map[string]any{
					"component": "bbb",
				},
			},
		},
		{
			desc:     "cannot overlay pingcap.com/xxx labels",
			isCreate: true,
			current: map[string]any{
				"labels": map[string]any{
					"pingcap.com/xxx": "bbb",
				},
			},
			wantErrs: []string{
				`spec.overlay.pod.metadata: Invalid value: "object": cannot overlay pod labels starting with 'pingcap.com/'`,
			},
		},
		{
			desc:     "ok to overlay pingcap.com label",
			isCreate: true,
			current: map[string]any{
				"labels": map[string]any{
					"pingcap.com": "bbb",
				},
			},
		},
	}

	return cases
}

func FeatureGates() []Case {
	cases := []Case{
		{
			desc:     "create with empty feature gates",
			isCreate: true,
			current:  []any{},
		},
		{
			desc:     "create with any feature gates",
			isCreate: true,
			current: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
			},
		},
		{
			desc:     "create with FeatureModification",
			isCreate: true,
			current: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
		},
		{
			desc: "can update feature gates if FeatureModification is enabled",
			old: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
			current: []any{
				map[string]any{
					"name": "DisablePDDefaultReadinessProbe",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
		},
		{
			desc: "FeatureModification remains enabled, other features removed",
			old: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
			current: []any{
				map[string]any{
					"name": "FeatureModification",
				},
			},
		},
		{
			desc: "FeatureModification remains enabled, other features added",
			old: []any{
				map[string]any{
					"name": "FeatureModification",
				},
			},
			current: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
		},
		{
			desc: "cannot update FeatureGates if FeatureModification is not enabled",
			old: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
			},
			current: []any{
				map[string]any{
					"name": "DisablePDDefaultReadinessProbe",
				},
			},
			wantErrs: []string{
				`spec.featureGates: Invalid value: "array": can only enable FeatureModification if it's not enabled`,
			},
		},
		{
			desc: "can only enable FeatureModification if FeatureModification is not enabled",
			old: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
			},
			current: []any{
				map[string]any{
					"name": "DisablePDDefaultReadinessProbe",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
			wantErrs: []string{
				`spec.featureGates: Invalid value: "array": can only enable FeatureModification if it's not enabled`,
			},
		},
		{
			desc: "cannot disable FeatureModification",
			old: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
			current: []any{
				map[string]any{
					"name": "VolumeAttributeClass",
				},
			},
			wantErrs: []string{
				`spec.featureGates: Invalid value: "array": cannot disable FeatureModification`,
			},
		},
	}

	return cases
}
