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
	"fmt"
	"strings"
)

const (
	requireDataVolumeErrMsg = `spec: Invalid value: "object": data volume must be configured`

	groupNameLengthLimit    = 40
	instanceNameLengthLimit = 47
	clusterNameLengthLimit  = 37
)

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
			desc:    "update with empty feature gates",
			old:     []any{},
			current: []any{},
		},
		{
			desc:     "create with any feature gates",
			isCreate: true,
			current: []any{
				map[string]any{
					"name": "VolumeAttributesClass",
				},
			},
		},
		{
			desc: "update without any update",
			old: []any{
				map[string]any{
					"name": "VolumeAttributesClass",
				},
			},
			current: []any{
				map[string]any{
					"name": "VolumeAttributesClass",
				},
			},
		},
		{
			desc:     "create with FeatureModification",
			isCreate: true,
			current: []any{
				map[string]any{
					"name": "VolumeAttributesClass",
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
					"name": "VolumeAttributesClass",
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
					"name": "VolumeAttributesClass",
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
					"name": "VolumeAttributesClass",
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
					"name": "VolumeAttributesClass",
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
					"name": "VolumeAttributesClass",
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
					"name": "VolumeAttributesClass",
				},
				map[string]any{
					"name": "FeatureModification",
				},
			},
			current: []any{
				map[string]any{
					"name": "VolumeAttributesClass",
				},
			},
			wantErrs: []string{
				`spec.featureGates: Invalid value: "array": cannot disable FeatureModification`,
			},
		},
	}

	return cases
}

func ServerLabels() []Case {
	errMsg := "spec.server.labels: Invalid value: \"object\": labels cannot contain 'host', 'region', or 'zone' keys"

	return []Case{
		{
			desc:     "set host label",
			isCreate: true,
			current: map[string]any{
				"host": "foo",
			},
			wantErrs: []string{errMsg},
		},
		{
			desc:     "set region label",
			isCreate: true,
			current: map[string]any{
				"region": "foo",
			},
			wantErrs: []string{errMsg},
		},
		{
			desc:     "set zone label",
			isCreate: true,
			current: map[string]any{
				"zone": "foo",
			},
			wantErrs: []string{errMsg},
		},
		{
			desc:     "set custom label",
			isCreate: true,
			current: map[string]any{
				"foo": "bar",
			},
		},
	}
}

func OverlayVolumeClaims(requireDataVolume bool) []Case {
	errMsg := `spec: Invalid value: "object": overlay volumeClaims names must exist in volumes`

	baseSpec := map[string]any{
		"cluster": map[string]any{
			"name": "test",
		},
		"subdomain": "test",
		"version":   "v8.1.0",
	}

	wantErrsFunc := func(requireDataVolume bool) []string {
		if requireDataVolume {
			return []string{errMsg, requireDataVolumeErrMsg}
		}
		return []string{errMsg}
	}

	cases := []Case{
		{
			desc:     "no overlay is ok",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
				}
				return spec
			}(),
		},
		{
			desc:     "overlay without volumeClaims is ok",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
				}
				spec["overlay"] = map[string]any{
					"pod": map[string]any{
						"metadata": map[string]any{
							"labels": map[string]any{
								"test": "value",
							},
						},
					},
				}
				return spec
			}(),
		},
		{
			desc:     "overlay volumeClaims with matching volume name is ok",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
				}
				spec["overlay"] = map[string]any{
					"volumeClaims": []any{
						map[string]any{
							"name": "data",
							"volumeClaim": map[string]any{
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "value",
									},
								},
							},
						},
					},
				}
				return spec
			}(),
		},
		{
			desc:     "overlay volumeClaims with multiple matching volume names is ok",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
					map[string]any{
						"name":    "log",
						"mounts":  []any{map[string]any{"type": "log"}},
						"storage": "10Gi",
					},
				}
				spec["overlay"] = map[string]any{
					"volumeClaims": []any{
						map[string]any{
							"name": "data",
							"volumeClaim": map[string]any{
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "value",
									},
								},
							},
						},
						map[string]any{
							"name": "log",
							"volumeClaim": map[string]any{
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "value",
									},
								},
							},
						},
					},
				}
				return spec
			}(),
		},
		{
			desc:     "overlay volumeClaims with non-matching volume name fails",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
				}
				spec["overlay"] = map[string]any{
					"volumeClaims": []any{
						map[string]any{
							"name": "log",
							"volumeClaim": map[string]any{
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "value",
									},
								},
							},
						},
					},
				}
				return spec
			}(),
			wantErrs: []string{errMsg},
		},
		{
			desc:     "overlay volumeClaims with mixed matching and non-matching volume names fails",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
				}
				spec["overlay"] = map[string]any{
					"volumeClaims": []any{
						map[string]any{
							"name": "data",
							"volumeClaim": map[string]any{
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "value",
									},
								},
							},
						},
						map[string]any{
							"name": "nonexistent",
							"volumeClaim": map[string]any{
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "value",
									},
								},
							},
						},
					},
				}
				return spec
			}(),
			wantErrs: []string{errMsg},
		},
		{
			desc:     "empty volumes with overlay volumeClaims fails",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{}
				spec["overlay"] = map[string]any{
					"volumeClaims": []any{
						map[string]any{
							"name": "data",
							"volumeClaim": map[string]any{
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "value",
									},
								},
							},
						},
					},
				}
				return spec
			}(),
			wantErrs: wantErrsFunc(requireDataVolume),
		},
	}

	return cases
}

func DataVolumeRequired() []Case {
	baseSpec := map[string]any{
		"cluster": map[string]any{
			"name": "test",
		},
		"subdomain": "test",
		"version":   "v8.1.0",
	}

	cases := []Case{
		{
			desc:     "data volume is configured correctly",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
				}
				return spec
			}(),
		},
		{
			desc:     "data volume with other volumes is ok",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "data",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
					map[string]any{
						"name":    "log",
						"mounts":  []any{map[string]any{"type": "log"}},
						"storage": "10Gi",
					},
				}
				return spec
			}(),
		},
		{
			desc:     "missing volumes field fails",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				return spec
			}(),
			wantErrs: []string{"spec.volumes: Required value"},
		},
		{
			desc:     "empty volumes array fails",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{}
				return spec
			}(),
			wantErrs: []string{requireDataVolumeErrMsg},
		},
		{
			desc:     "volumes without data volume fails",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "log",
						"mounts":  []any{map[string]any{"type": "log"}},
						"storage": "10Gi",
					},
				}
				return spec
			}(),
			wantErrs: []string{requireDataVolumeErrMsg},
		},
		{
			desc:     "volumes with differently named volume fails",
			isCreate: true,
			current: func() map[string]any {
				spec := make(map[string]any)
				for k, v := range baseSpec {
					spec[k] = v
				}
				spec["volumes"] = []any{
					map[string]any{
						"name":    "storage",
						"mounts":  []any{map[string]any{"type": "data"}},
						"storage": "20Gi",
					},
				}
				return spec
			}(),
			wantErrs: []string{requireDataVolumeErrMsg},
		},
	}

	return cases
}

func VolumeAttributesClassNameValidation() []Case {
	errMsg := "spec.volumes[0]: Invalid value: \"object\": VolumeAttributesClassName cannot be changed from non-nil to nil"

	cases := []Case{
		{
			desc:     "create with nil volumeAttributesClassName is ok",
			isCreate: true,
			current: []any{
				map[string]any{
					"name":    "data",
					"mounts":  []any{map[string]any{"type": "data"}},
					"storage": "20Gi",
				},
			},
		},
		{
			desc:     "create with volumeAttributesClassName is ok",
			isCreate: true,
			current: []any{
				map[string]any{
					"name":                      "data",
					"mounts":                    []any{map[string]any{"type": "data"}},
					"storage":                   "20Gi",
					"volumeAttributesClassName": "test-class",
				},
			},
		},
		{
			desc:     "update from nil to non-nil volumeAttributesClassName is ok",
			isCreate: false,
			old: []any{
				map[string]any{
					"name":    "data",
					"mounts":  []any{map[string]any{"type": "data"}},
					"storage": "20Gi",
				},
			},
			current: []any{
				map[string]any{
					"name":                      "data",
					"mounts":                    []any{map[string]any{"type": "data"}},
					"storage":                   "20Gi",
					"volumeAttributesClassName": "test-class",
				},
			},
		},
		{
			desc:     "update from non-nil to different non-nil volumeAttributesClassName is ok",
			isCreate: false,
			old: []any{
				map[string]any{
					"name":                      "data",
					"mounts":                    []any{map[string]any{"type": "data"}},
					"storage":                   "20Gi",
					"volumeAttributesClassName": "old-class",
				},
			},
			current: []any{
				map[string]any{
					"name":                      "data",
					"mounts":                    []any{map[string]any{"type": "data"}},
					"storage":                   "20Gi",
					"volumeAttributesClassName": "new-class",
				},
			},
		},
		{
			desc:     "update from non-nil to nil volumeAttributesClassName should fail",
			isCreate: false,
			old: []any{
				map[string]any{
					"name":                      "data",
					"mounts":                    []any{map[string]any{"type": "data"}},
					"storage":                   "20Gi",
					"volumeAttributesClassName": "test-class",
				},
			},
			current: []any{
				map[string]any{
					"name":    "data",
					"mounts":  []any{map[string]any{"type": "data"}},
					"storage": "20Gi",
				},
			},
			wantErrs: []string{errMsg},
		},
	}
	return cases
}

func Version() []Case {
	// Valid semantic versions
	validVersions := []string{
		"0.0.4",
		"1.2.3",
		"10.20.30",
		"1.1.2-prerelease+meta",
		"1.1.2+meta",
		"1.1.2+meta-valid",
		"1.0.0-alpha",
		"1.0.0-beta",
		"1.0.0-alpha.beta",
		"1.0.0-alpha.beta.1",
		"1.0.0-alpha.1",
		"1.0.0-alpha0.valid",
		"1.0.0-alpha.0valid",
		"1.0.0-alpha-a.b-c-somethinglong+build.1-aef.1-its-okay",
		"1.0.0-rc.1+build.1",
		"2.0.0-rc.1+build.123",
		"1.2.3-beta",
		"10.2.3-DEV-SNAPSHOT",
		"1.2.3-SNAPSHOT-123",
		"1.0.0",
		"2.0.0",
		"1.1.7",
		"2.0.0+build.1848",
		"2.0.1-alpha.1227",
		"1.0.0-alpha+beta",
		"1.2.3----RC-SNAPSHOT.12.9.1--.12+788",
		"1.2.3----R-S.12.9.1--.12+meta",
		"1.2.3----RC-SNAPSHOT.12.9.1--.12",
		"1.0.0+0.build.1-rc.10000aaa-kk-0.1",
		"99999999999999999999999.999999999999999999.99999999999999999",
		"1.0.0-0A.is.legal",
		// Versions with v prefix
		"v0.0.4",
		"v1.2.3",
		"v10.20.30",
		"v1.1.2-prerelease+meta",
		"v1.0.0-alpha",
		"v2.0.0+build.1848",
		"v8.1.0",
	}

	// Invalid semantic versions
	invalidVersions := []string{
		"1",
		"1.2",
		"1.2.3-0123",
		"1.2.3-0123.0123",
		"1.1.2+.123",
		"+invalid",
		"-invalid",
		"-invalid+invalid",
		"-invalid.01",
		"alpha",
		"alpha.beta",
		"alpha.beta.1",
		"alpha.1",
		"alpha+beta",
		"alpha_beta",
		"alpha.",
		"alpha..",
		"beta",
		"1.0.0-alpha_beta",
		"-alpha.",
		"1.0.0-alpha..",
		"1.0.0-alpha..1",
		"1.0.0-alpha...1",
		"1.0.0-alpha....1",
		"1.0.0-alpha.....1",
		"1.0.0-alpha......1",
		"1.0.0-alpha.......1",
		"01.1.1",
		"1.01.1",
		"1.1.01",
		"1.2",
		"1.2.3.DEV",
		"1.2-SNAPSHOT",
		"1.2.31.2.3----RC-SNAPSHOT.12.09.1--..12+788",
		"1.2-RC-SNAPSHOT",
		"-1.0.3-gamma+b7718",
		"+justmeta",
		"9.8.7+meta+meta",
		"9.8.7-whatever+meta+meta",
		"99999999999999999999999.999999999999999999.99999999999999999----RC-SNAPSHOT.12.09.1--------------------------------..12",
		"",
		"  ",
		"vv1.2.3",
		"1.2.3v",
		"1.2.3-",
		"1.2.3+",
	}

	var cases []Case

	// Add test cases for valid versions
	for _, version := range validVersions {
		cases = append(cases, Case{
			desc:     fmt.Sprintf("version '%s' is valid semantic version", version),
			isCreate: true,
			current:  version,
		})
	}

	// Add test cases for invalid versions
	for _, version := range invalidVersions {
		cases = append(cases, Case{
			desc:     fmt.Sprintf("version '%s' is invalid semantic version", version),
			isCreate: true,
			current:  version,
			wantErrs: []string{
				fmt.Sprintf(`spec.version: Invalid value: "%s": spec.version in body should match '^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$'`, version),
			},
		})
	}

	return cases
}

func NameLength(limit int) []Case {
	cases := []Case{
		{
			desc:     fmt.Sprintf("name length is %d", limit),
			isCreate: true,
			current:  strings.Repeat("a", limit),
		},
		{
			desc:     "name is too long",
			isCreate: true,
			current:  strings.Repeat("a", limit+1),
			wantErrs: []string{
				fmt.Sprintf("<nil>: Invalid value: \"object\": name must not exceed %d characters", limit),
			},
		},
	}

	return cases
}
