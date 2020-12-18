// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package autoscaler

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestCheckStsAutoScalingInterval(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                  string
		group                 string
		memberType            v1alpha1.MemberType
		HaveScaled            bool
		LastScaleIntervalSec  int
		expectedPermitScaling bool
	}{
		{
			name:                  "tikv, first scaling",
			group:                 "group",
			memberType:            v1alpha1.TiKVMemberType,
			HaveScaled:            false,
			LastScaleIntervalSec:  0,
			expectedPermitScaling: true,
		},
		{
			name:                  "tikv, scaling 60 secs ago",
			group:                 "group",
			memberType:            v1alpha1.TiKVMemberType,
			HaveScaled:            true,
			LastScaleIntervalSec:  60,
			expectedPermitScaling: false,
		},
		{
			name:                  "tidb, first scaling",
			group:                 "group",
			memberType:            v1alpha1.TiDBMemberType,
			HaveScaled:            false,
			LastScaleIntervalSec:  0,
			expectedPermitScaling: true,
		},
		{
			name:                  "tidb, scaling 60 secs ago",
			group:                 "group",
			memberType:            v1alpha1.TiDBMemberType,
			HaveScaled:            true,
			LastScaleIntervalSec:  60,
			expectedPermitScaling: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tac := newTidbClusterAutoScaler()
			tac.Status.TiKV = map[string]v1alpha1.TikvAutoScalerStatus{}
			tac.Status.TiDB = map[string]v1alpha1.TidbAutoScalerStatus{}
			intervalSec := int32(100)
			if tt.memberType == v1alpha1.TiKVMemberType {
				if tt.HaveScaled {
					d := time.Duration(tt.LastScaleIntervalSec) * time.Second
					tac.Status.TiKV[tt.group] = v1alpha1.TikvAutoScalerStatus{
						BasicAutoScalerStatus: v1alpha1.BasicAutoScalerStatus{
							LastAutoScalingTimestamp: &metav1.Time{
								Time: time.Now().Truncate(d),
							},
						},
					}
				}
			} else if tt.memberType == v1alpha1.TiDBMemberType {
				if tt.HaveScaled {
					d := time.Duration(tt.LastScaleIntervalSec) * time.Second
					tac.Status.TiDB[tt.group] = v1alpha1.TidbAutoScalerStatus{
						BasicAutoScalerStatus: v1alpha1.BasicAutoScalerStatus{
							LastAutoScalingTimestamp: &metav1.Time{
								Time: time.Now().Truncate(d),
							},
						},
					}
				}
			}
			r := checkAutoScalingInterval(tac, intervalSec, tt.memberType, tt.group)
			g.Expect(r).Should(Equal(tt.expectedPermitScaling))
		})

	}
}

func TestDefaultTac(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	testcases := []struct {
		name      string
		sourceTac *v1alpha1.TidbClusterAutoScaler
		expectTac *v1alpha1.TidbClusterAutoScaler
	}{
		// case1
		{
			name: "source tac spec don't have resources",
			sourceTac: func() *v1alpha1.TidbClusterAutoScaler {
				sourceTAC := newTidbClusterAutoScaler()
				sourceTAC.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				sourceTAC.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				return sourceTAC
			}(),
			expectTac: func() *v1alpha1.TidbClusterAutoScaler {
				expectTac := newTidbClusterAutoScaler()
				expectTac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
					"default_tikv": {
						CPU:     tc.Spec.TiKV.Requests.Cpu().DeepCopy(),
						Memory:  tc.Spec.TiKV.Requests.Memory().DeepCopy(),
						Storage: tc.Spec.TiKV.Requests[corev1.ResourceStorage].DeepCopy(),
					},
				}
				expectTac.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
					"default_tidb": {
						CPU:    tc.Spec.TiDB.Requests.Cpu().DeepCopy(),
						Memory: tc.Spec.TiDB.Requests.Memory().DeepCopy(),
					},
				}
				expectTac.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"default_tikv"},
					},
				}
				expectTac.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"default_tidb"},
					},
				}
				return expectTac
			}(),
		},
		// case2
		{
			name: "source tac spec only have tidb",
			sourceTac: func() *v1alpha1.TidbClusterAutoScaler {
				sourceTAC := newTidbClusterAutoScaler()
				sourceTAC.Spec.TiKV = nil
				sourceTAC.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
					"case2": {
						CPU:    resource.MustParse("1"),
						Memory: resource.MustParse("1G"),
					},
				}
				sourceTAC.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				return sourceTAC
			}(),
			expectTac: func() *v1alpha1.TidbClusterAutoScaler {
				expectTac := newTidbClusterAutoScaler()
				expectTac.Spec.TiKV = nil
				expectTac.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
					"case2": {
						CPU:    resource.MustParse("1"),
						Memory: resource.MustParse("1G"),
					},
				}
				expectTac.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"case2"},
					},
				}
				return expectTac
			}(),
		},
		// case3
		{
			name: "source tac spec only have tikv",
			sourceTac: func() *v1alpha1.TidbClusterAutoScaler {
				sourceTAC := newTidbClusterAutoScaler()
				sourceTAC.Spec.TiDB = nil
				sourceTAC.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
					"case3": {
						CPU:     resource.MustParse("1"),
						Memory:  resource.MustParse("1G"),
						Storage: resource.MustParse("10G"),
					},
				}
				sourceTAC.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				return sourceTAC
			}(),
			expectTac: func() *v1alpha1.TidbClusterAutoScaler {
				expectTac := newTidbClusterAutoScaler()
				expectTac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
					"case3": {
						CPU:     resource.MustParse("1"),
						Memory:  resource.MustParse("1G"),
						Storage: resource.MustParse("10G"),
					},
				}
				expectTac.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"case3"},
					},
				}
				expectTac.Spec.TiDB = nil
				return expectTac
			}(),
		},
		// case4
		{
			name: "source tac spec only have tidb, no resources",
			sourceTac: func() *v1alpha1.TidbClusterAutoScaler {
				sourceTAC := newTidbClusterAutoScaler()
				sourceTAC.Spec.TiKV = nil
				sourceTAC.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				return sourceTAC
			}(),
			expectTac: func() *v1alpha1.TidbClusterAutoScaler {
				expectTac := newTidbClusterAutoScaler()
				expectTac.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
					"default_tidb": {
						CPU:    tc.Spec.TiDB.Requests.Cpu().DeepCopy(),
						Memory: tc.Spec.TiDB.Requests.Memory().DeepCopy(),
					},
				}
				expectTac.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"default_tidb"},
					},
				}
				expectTac.Spec.TiKV = nil
				return expectTac
			}(),
		},
		// case5
		{
			name: "source tac spec only have tikv, no resources",
			sourceTac: func() *v1alpha1.TidbClusterAutoScaler {
				sourceTAC := newTidbClusterAutoScaler()
				sourceTAC.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				sourceTAC.Spec.TiDB = nil
				return sourceTAC
			}(),
			expectTac: func() *v1alpha1.TidbClusterAutoScaler {
				expectTac := newTidbClusterAutoScaler()
				expectTac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
					"default_tikv": {
						CPU:     tc.Spec.TiKV.Requests.Cpu().DeepCopy(),
						Memory:  tc.Spec.TiKV.Requests.Memory().DeepCopy(),
						Storage: tc.Spec.TiKV.Requests[corev1.ResourceStorage].DeepCopy(),
					},
				}
				expectTac.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"default_tikv"},
					},
				}
				expectTac.Spec.TiDB = nil
				return expectTac
			}(),
		},
		// case6
		{
			name: "source tac spec don't have tikv/tidb",
			sourceTac: func() *v1alpha1.TidbClusterAutoScaler {
				sourceTAC := newTidbClusterAutoScaler()
				sourceTAC.Spec.TiDB = nil
				sourceTAC.Spec.TiKV = nil
				return sourceTAC
			}(),
			expectTac: func() *v1alpha1.TidbClusterAutoScaler {
				expectTac := newTidbClusterAutoScaler()
				expectTac.Spec.TiDB = nil
				expectTac.Spec.TiKV = nil
				return expectTac
			}(),
		},
		// case7
		{
			name: "source tac spec have self-defined resources",
			sourceTac: func() *v1alpha1.TidbClusterAutoScaler {
				sourceTAC := newTidbClusterAutoScaler()
				sourceTAC.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
					"storage": {
						CPU:     resource.MustParse("1"),
						Memory:  resource.MustParse("1G"),
						Storage: resource.MustParse("10G"),
					},
				}
				sourceTAC.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				sourceTAC.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
					"no-storage": {
						CPU:    resource.MustParse("1"),
						Memory: resource.MustParse("1G"),
					},
					"storage": {
						CPU:     resource.MustParse("1"),
						Memory:  resource.MustParse("1G"),
						Storage: resource.MustParse("10G"),
					},
				}
				sourceTAC.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold: 0.8,
					},
				}
				return sourceTAC
			}(),
			expectTac: func() *v1alpha1.TidbClusterAutoScaler {
				expectTac := newTidbClusterAutoScaler()
				expectTac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
					"storage": {
						CPU:     resource.MustParse("1"),
						Memory:  resource.MustParse("1G"),
						Storage: resource.MustParse("10G"),
					},
				}
				expectTac.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"storage"},
					},
				}
				expectTac.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
					"no-storage": {
						CPU:    resource.MustParse("1"),
						Memory: resource.MustParse("1G"),
					},
					"storage": {
						CPU:     resource.MustParse("1"),
						Memory:  resource.MustParse("1G"),
						Storage: resource.MustParse("10G"),
					},
				}
				expectTac.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
					corev1.ResourceCPU: {
						MaxThreshold:  0.8,
						MinThreshold:  pointer.Float64Ptr(0.1),
						ResourceTypes: []string{"no-storage", "storage"},
					},
				}
				return expectTac
			}(),
		},
	}
	for _, testcase := range testcases {
		t.Log(testcase.name)
		tc := newTidbCluster()
		defaultTAC(testcase.sourceTac, tc)
		if testcase.sourceTac.Spec.TiKV == nil {
			g.Expect(testcase.expectTac.Spec.TiKV).Should(BeNil())
		} else {
			g.Expect(testcase.sourceTac.Spec.TiKV.Resources).Should(Equal(testcase.expectTac.Spec.TiKV.Resources))
			g.Expect(testcase.sourceTac.Spec.TiKV.Rules).Should(Equal(testcase.expectTac.Spec.TiKV.Rules))
		}
		if testcase.sourceTac.Spec.TiDB == nil {
			g.Expect(testcase.expectTac.Spec.TiDB).Should(BeNil())
		} else {
			g.Expect(testcase.sourceTac.Spec.TiDB.Resources).Should(Equal(testcase.expectTac.Spec.TiDB.Resources))
			g.Expect(testcase.sourceTac.Spec.TiDB.Rules).Should(Equal(testcase.expectTac.Spec.TiDB.Rules))
		}
		g.Expect(validateTAC(testcase.sourceTac)).Should(BeNil())
	}
}

func TestAutoscalerToStrategy(t *testing.T) {
	g := NewGomegaWithT(t)
	tac := newTidbClusterAutoScaler()
	tac.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
		"compute": {
			CPU:    resource.MustParse("8000m"),
			Memory: resource.MustParse("16Gi"),
		},
	}
	tac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
		"storage": {
			CPU:     resource.MustParse("2000m"),
			Memory:  resource.MustParse("4Gi"),
			Storage: resource.MustParse("2000Gi"),
			Count:   pointer.Int32Ptr(4),
		},
	}
	tac.Spec.TiDB.Rules = make(map[corev1.ResourceName]v1alpha1.AutoRule)
	tac.Spec.TiKV.Rules = make(map[corev1.ResourceName]v1alpha1.AutoRule)
	tac.Spec.TiDB.Rules[corev1.ResourceCPU] = v1alpha1.AutoRule{
		MaxThreshold:  0.8,
		MinThreshold:  pointer.Float64Ptr(0.2),
		ResourceTypes: []string{"resource_a"},
	}
	tac.Spec.TiKV.Rules[corev1.ResourceCPU] = v1alpha1.AutoRule{
		MaxThreshold:  0.8,
		MinThreshold:  pointer.Float64Ptr(0.2),
		ResourceTypes: []string{"resource_a", "resource_b"},
	}
	tidbStrategy := autoscalerToStrategy(tac, v1alpha1.TiDBMemberType)
	g.Expect(len(tidbStrategy.Resources)).Should(Equal(1))
	g.Expect(len(tidbStrategy.Rules)).Should(Equal(1))

	tikvStrategy := autoscalerToStrategy(tac, v1alpha1.TiKVMemberType)
	g.Expect(len(tikvStrategy.Resources)).Should(Equal(1))
	g.Expect(len(tikvStrategy.Rules)).Should(Equal(1))
}

func TestValidateTidbClusterAutoScaler(t *testing.T) {
	g := NewGomegaWithT(t)
	minThreshold := 0.1
	invalidMinThreshold := 2.0

	tac := newTidbClusterAutoScaler()
	tac.Spec.TiKV = nil
	tac.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
		"compute": {
			Memory: resource.MustParse("2Gi"),
			CPU:    resource.MustParse("1000m"),
		},
	}

	// Case 1: Invalid max_threshold
	tac.Spec.TiDB.BasicAutoScalerSpec.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold:  2,
			MinThreshold:  &minThreshold,
			ResourceTypes: []string{"compute"},
		},
	}
	err := validateTAC(tac)
	g.Expect(err).Should(MatchError(fmt.Errorf("max_threshold (%v) should be between 0 and 1 for rule cpu of tidb in %s/%s", 2, tac.Namespace, tac.Name)))

	// Case 2: Invalid min_theshold
	tac.Spec.TiDB.BasicAutoScalerSpec.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold:  1,
			MinThreshold:  &invalidMinThreshold,
			ResourceTypes: []string{"compute"},
		},
	}
	err = validateTAC(tac)
	g.Expect(err).Should(MatchError(fmt.Errorf("min_threshold (%v) should be between 0 and 1 for rule cpu of tidb in %s/%s", invalidMinThreshold, tac.Namespace, tac.Name)))

	// Case 3: No resources
	tac.Spec.TiDB.BasicAutoScalerSpec.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold: 1,
			MinThreshold: &minThreshold,
		},
	}
	err = validateTAC(tac)
	g.Expect(err).Should(MatchError(fmt.Errorf("no resources provided for rule cpu of tidb in %s/%s", tac.Namespace, tac.Name)))

	// Case 4: Resource not in Spec
	tac.Spec.TiDB.BasicAutoScalerSpec.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold:  1,
			MinThreshold:  &minThreshold,
			ResourceTypes: []string{"non_exist"},
		},
	}
	err = validateTAC(tac)
	g.Expect(err).Should(MatchError(fmt.Errorf("unknown resource non_exist for tidb in %s/%s", tac.Namespace, tac.Name)))

	// Case 5: min_threshold > max_threshold for cpu rule
	tac.Spec.TiDB.BasicAutoScalerSpec.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold:  0.05,
			MinThreshold:  &minThreshold,
			ResourceTypes: []string{"compute"},
		},
	}
	err = validateTAC(tac)
	g.Expect(err).Should(MatchError(fmt.Errorf("min_threshold (%v) > max_threshold (%v) for cpu rule of tidb in %s/%s", minThreshold, 0.05, tac.Namespace, tac.Name)))

	// Case 6: Resource does not have storage for tikv
	tac.Spec.TiDB = nil
	tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{
		BasicAutoScalerSpec: v1alpha1.BasicAutoScalerSpec{
			Rules: map[corev1.ResourceName]v1alpha1.AutoRule{
				corev1.ResourceCPU: {
					MaxThreshold:  0.8,
					ResourceTypes: []string{"compute"},
				},
			},
		},
	}
	tac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
		"compute": {
			Memory: resource.MustParse("2Gi"),
			CPU:    resource.MustParse("1000m"),
		},
	}
	err = validateTAC(tac)
	g.Expect(err).Should(MatchError(fmt.Errorf("resource compute defined for tikv does not have storage in %s/%s", tac.Namespace, tac.Name)))

	// Case 7: Valid spec
	tac = newTidbClusterAutoScaler()

	tac.Spec.TiDB.BasicAutoScalerSpec = v1alpha1.BasicAutoScalerSpec{
		Rules: map[corev1.ResourceName]v1alpha1.AutoRule{
			corev1.ResourceCPU: {
				MaxThreshold:  0.8,
				MinThreshold:  &minThreshold,
				ResourceTypes: []string{"compute"},
			},
		},
	}
	tac.Spec.TiKV.BasicAutoScalerSpec = v1alpha1.BasicAutoScalerSpec{
		Rules: map[corev1.ResourceName]v1alpha1.AutoRule{
			corev1.ResourceCPU: {
				MaxThreshold:  0.8,
				MinThreshold:  &minThreshold,
				ResourceTypes: []string{"storage"},
			},
		},
	}
	tac.Spec.TiDB.Resources = map[string]v1alpha1.AutoResource{
		"compute": {
			Memory: resource.MustParse("2Gi"),
			CPU:    resource.MustParse("1000m"),
		},
	}
	tac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
		"storage": {
			Memory:  resource.MustParse("2Gi"),
			CPU:     resource.MustParse("1000m"),
			Storage: resource.MustParse("1000Gi"),
		},
	}
	err = validateTAC(tac)
	g.Expect(err).Should(BeNil())
}

func newTidbClusterAutoScaler() *v1alpha1.TidbClusterAutoScaler {
	tac := &v1alpha1.TidbClusterAutoScaler{}
	tac.Name = "tac"
	tac.Namespace = "default"
	tac.Annotations = map[string]string{}
	tac.Spec.Cluster = v1alpha1.TidbClusterRef{
		Name:      "tc",
		Namespace: "default",
	}
	tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{}
	tac.Spec.TiDB = &v1alpha1.TidbAutoScalerSpec{}
	return tac
}

func newTidbCluster() *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{}
	tc.Name = "tc"
	tc.Namespace = "default"
	computeResource := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1000m"),
		corev1.ResourceMemory: resource.MustParse("2Gi"),
	}
	storageResource := corev1.ResourceList{
		corev1.ResourceCPU:     resource.MustParse("1000m"),
		corev1.ResourceMemory:  resource.MustParse("2Gi"),
		corev1.ResourceStorage: resource.MustParse("1000Gi"),
	}
	tc.Spec.TiDB = &v1alpha1.TiDBSpec{
		ResourceRequirements: corev1.ResourceRequirements{
			Requests: computeResource,
			Limits:   computeResource,
		},
	}
	tc.Spec.TiKV = &v1alpha1.TiKVSpec{
		ResourceRequirements: corev1.ResourceRequirements{
			Requests: storageResource,
			Limits:   storageResource,
		},
	}
	return tc
}
