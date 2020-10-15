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
	"github.com/pingcap/tidb-operator/pkg/label"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

func TestCheckStsAutoScalingInterval(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                  string
		memberType            v1alpha1.MemberType
		HaveScaled            bool
		LastScaleIntervalSec  int
		expectedPermitScaling bool
	}{
		{
			name:                  "tikv, first scaling",
			memberType:            v1alpha1.TiKVMemberType,
			HaveScaled:            false,
			LastScaleIntervalSec:  0,
			expectedPermitScaling: true,
		},
		{
			name:                  "tikv, scaling 60 secs ago",
			memberType:            v1alpha1.TiKVMemberType,
			HaveScaled:            true,
			LastScaleIntervalSec:  60,
			expectedPermitScaling: false,
		},
		{
			name:                  "tidb, first scaling",
			memberType:            v1alpha1.TiDBMemberType,
			HaveScaled:            false,
			LastScaleIntervalSec:  0,
			expectedPermitScaling: true,
		},
		{
			name:                  "tidb, scaling 60 secs ago",
			memberType:            v1alpha1.TiDBMemberType,
			HaveScaled:            true,
			LastScaleIntervalSec:  60,
			expectedPermitScaling: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tac := newTidbClusterAutoScaler()
			intervalSec := int32(100)
			if tt.memberType == v1alpha1.TiKVMemberType {
				if !tt.HaveScaled {
					tac.Annotations = map[string]string{}
				} else {
					d := time.Duration(tt.LastScaleIntervalSec) * time.Second
					tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Truncate(d).Unix())
				}
			} else if tt.memberType == v1alpha1.TiDBMemberType {
				if !tt.HaveScaled {
					tac.Annotations = map[string]string{}
				} else {
					d := time.Duration(tt.LastScaleIntervalSec) * time.Second
					tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Truncate(d).Unix())
				}
			}
			r, err := checkStsAutoScalingInterval(tac, intervalSec, tt.memberType)
			g.Expect(err).Should(BeNil())
			g.Expect(r).Should(Equal(tt.expectedPermitScaling))
		})

	}
}

func TestCheckStsAutoScalingPrerequisites(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                string
		stsUpdating         bool
		stsScaling          bool
		expectedCheckResult bool
	}{
		{
			name:                "upgrading",
			stsUpdating:         true,
			stsScaling:          false,
			expectedCheckResult: false,
		},
		{
			name:                "scaling",
			stsUpdating:         false,
			stsScaling:          true,
			expectedCheckResult: false,
		},
		{
			name:                "no upgrading, no scaling",
			stsUpdating:         false,
			stsScaling:          false,
			expectedCheckResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts := newSts()
			if tt.stsUpdating {
				sts.Status.UpdateRevision = "1"
				sts.Status.CurrentRevision = "2"
			} else {
				sts.Status.UpdateRevision = "1"
				sts.Status.CurrentRevision = "1"
			}
			if tt.stsScaling {
				sts.Spec.Replicas = pointer.Int32Ptr(1)
				sts.Status.Replicas = 2
			} else {
				sts.Spec.Replicas = pointer.Int32Ptr(1)
				sts.Status.Replicas = 1
			}
			r := checkStsAutoScalingPrerequisites(sts)
			g.Expect(r).Should(Equal(tt.expectedCheckResult))
		})
	}

}

func TestDefaultTac(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	tac := newTidbClusterAutoScaler()
	tac.Spec.TiDB = nil
	tac.Spec.TiKV.ScaleOutIntervalSeconds = nil
	tac.Spec.TiKV.ScaleInIntervalSeconds = nil
	tac.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold: 0.8,
		},
	}
	defaultTAC(tac, tc)
	g.Expect(*tac.Spec.TiKV.ScaleOutIntervalSeconds).Should(Equal(int32(300)))
	g.Expect(*tac.Spec.TiKV.ScaleInIntervalSeconds).Should(Equal(int32(500)))
	g.Expect(tac.Spec.TiKV.Resources).Should(Equal(map[string]v1alpha1.AutoResource{
		"default_tikv": {
			CPU:     tc.Spec.TiKV.Requests.Cpu().DeepCopy(),
			Memory:  tc.Spec.TiKV.Requests.Memory().DeepCopy(),
			Storage: tc.Spec.TiKV.Requests[corev1.ResourceStorage].DeepCopy(),
		},
	}))
	g.Expect(*tac.Spec.TiKV.Rules[corev1.ResourceCPU].MinThreshold).Should(BeNumerically("==", 0.1))
	g.Expect(tac.Spec.TiKV.Rules[corev1.ResourceCPU].ResourceTypes).Should(ConsistOf([]string{"default_tikv"}))

	tac = newTidbClusterAutoScaler()
	tac.Spec.TiKV = nil
	tac.Spec.TiDB.ScaleOutIntervalSeconds = nil
	tac.Spec.TiDB.ScaleInIntervalSeconds = nil
	tac.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold: 0.8,
		},
	}
	defaultTAC(tac, tc)
	g.Expect(*tac.Spec.TiDB.ScaleOutIntervalSeconds).Should(Equal(int32(300)))
	g.Expect(*tac.Spec.TiDB.ScaleInIntervalSeconds).Should(Equal(int32(500)))
	g.Expect(tac.Spec.TiDB.Resources).Should(Equal(map[string]v1alpha1.AutoResource{
		"default_tidb": {
			CPU:     tc.Spec.TiDB.Requests.Cpu().DeepCopy(),
			Memory:  tc.Spec.TiDB.Requests.Memory().DeepCopy(),
			Storage: tc.Spec.TiDB.Requests[corev1.ResourceStorage].DeepCopy(),
		},
	}))
	g.Expect(*tac.Spec.TiDB.Rules[corev1.ResourceCPU].MinThreshold).Should(BeNumerically("==", 0.1))
	g.Expect(tac.Spec.TiDB.Rules[corev1.ResourceCPU].ResourceTypes).Should(ConsistOf([]string{"default_tidb"}))

	tac = newTidbClusterAutoScaler()
	tac.Spec.TiDB = nil
	tac.Spec.TiKV.ScaleOutIntervalSeconds = nil
	tac.Spec.TiKV.ScaleInIntervalSeconds = nil
	tac.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceStorage: {
			MaxThreshold: 0.8,
		},
	}
	tac.Spec.TiKV.Resources = map[string]v1alpha1.AutoResource{
		"storage": {
			CPU:     resource.MustParse("1000m"),
			Memory:  resource.MustParse("2Gi"),
			Storage: resource.MustParse("200Gi"),
		},
	}
	defaultTAC(tac, tc)
	g.Expect(tac.Spec.TiKV.Rules[corev1.ResourceStorage].ResourceTypes).Should(ConsistOf([]string{"storage"}))

	tac = newTidbClusterAutoScaler()
	tac.Spec.TiDB.ScaleOutIntervalSeconds = nil
	tac.Spec.TiDB.ScaleInIntervalSeconds = nil
	tac.Spec.TiKV.ScaleOutIntervalSeconds = nil
	tac.Spec.TiKV.ScaleInIntervalSeconds = nil
	tac.Spec.TiDB.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceCPU: {
			MaxThreshold: 0.8,
		},
	}
	tac.Spec.TiKV.Rules = map[corev1.ResourceName]v1alpha1.AutoRule{
		corev1.ResourceStorage: {
			MaxThreshold: 0.8,
		},
	}
	defaultTAC(tac, tc)
	g.Expect(tac.Spec.TiKV.Rules[corev1.ResourceStorage].ResourceTypes).Should(ConsistOf([]string{"default_tikv"}))
	g.Expect(tac.Spec.TiDB.Rules[corev1.ResourceCPU].ResourceTypes).Should(ConsistOf([]string{"default_tidb"}))
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

func newSts() *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(1),
		},
		Status: appsv1.StatefulSetStatus{
			CurrentRevision: "1",
			UpdateRevision:  "2",
			Replicas:        2,
		},
	}
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
