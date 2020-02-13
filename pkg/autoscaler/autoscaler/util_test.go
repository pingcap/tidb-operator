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
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
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

func TestLimitTargetReplicas(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name             string
		targetReplicas   int32
		minReplicas      int32
		maxReplicas      int32
		memberType       v1alpha1.MemberType
		expectedReplicas int32
	}{
		{
			name:             "tikv,smaller than min",
			targetReplicas:   1,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiKVMemberType,
			expectedReplicas: 2,
		},
		{
			name:             "tikv,equal min",
			targetReplicas:   2,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiKVMemberType,
			expectedReplicas: 2,
		},
		{
			name:             "tikv,bigger than min, smaller than max",
			targetReplicas:   3,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiKVMemberType,
			expectedReplicas: 3,
		},
		{
			name:             "tikv,equal max",
			targetReplicas:   4,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiKVMemberType,
			expectedReplicas: 4,
		},
		{
			name:             "tikv,greater than max",
			targetReplicas:   5,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiKVMemberType,
			expectedReplicas: 4,
		},
		//tidb
		{
			name:             "tidb,smaller than min",
			targetReplicas:   1,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiDBMemberType,
			expectedReplicas: 2,
		},
		{
			name:             "tidb,equal min",
			targetReplicas:   2,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiDBMemberType,
			expectedReplicas: 2,
		},
		{
			name:             "tidb,bigger than min, smaller than max",
			targetReplicas:   3,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiDBMemberType,
			expectedReplicas: 3,
		},
		{
			name:             "tidb,equal max",
			targetReplicas:   4,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiDBMemberType,
			expectedReplicas: 4,
		},
		{
			name:             "tidb,greater than max",
			targetReplicas:   5,
			minReplicas:      2,
			maxReplicas:      4,
			memberType:       v1alpha1.TiDBMemberType,
			expectedReplicas: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tac := newTidbClusterAutoScaler()
			if tt.memberType == v1alpha1.TiKVMemberType {
				tac.Spec.TiKV.MinReplicas = pointer.Int32Ptr(tt.minReplicas)
				tac.Spec.TiKV.MaxReplicas = tt.maxReplicas
			} else if tt.memberType == v1alpha1.TiDBMemberType {
				tac.Spec.TiDB.MinReplicas = pointer.Int32Ptr(tt.minReplicas)
				tac.Spec.TiDB.MaxReplicas = tt.maxReplicas
			}
			r := limitTargetReplicas(tt.targetReplicas, tac, tt.memberType)
			g.Expect(tt.expectedReplicas).Should(Equal(r))
		})
	}
}

func TestDefaultTac(t *testing.T) {
	g := NewGomegaWithT(t)
	tac := newTidbClusterAutoScaler()
	tac.Spec.TiDB = nil
	tac.Spec.TiKV.MinReplicas = nil
	tac.Spec.TiKV.Metrics = []autoscalingv2beta2.MetricSpec{}
	tac.Spec.TiKV.MetricsTimeDuration = nil
	tac.Spec.TiKV.ScaleOutIntervalSeconds = nil
	tac.Spec.TiKV.ScaleInIntervalSeconds = nil
	defaultTAC(tac)
	g.Expect(*tac.Spec.TiKV.MinReplicas).Should(Equal(int32(1)))
	g.Expect(len(tac.Spec.TiKV.Metrics)).Should(Equal(1))
	g.Expect(*tac.Spec.TiKV.MetricsTimeDuration).Should(Equal("3m"))
	g.Expect(*tac.Spec.TiKV.ScaleOutIntervalSeconds).Should(Equal(int32(300)))
	g.Expect(*tac.Spec.TiKV.ScaleInIntervalSeconds).Should(Equal(int32(500)))

	tac = newTidbClusterAutoScaler()
	tac.Spec.TiKV = nil
	tac.Spec.TiDB.MinReplicas = nil
	tac.Spec.TiDB.Metrics = []autoscalingv2beta2.MetricSpec{}
	tac.Spec.TiDB.MetricsTimeDuration = nil
	tac.Spec.TiDB.ScaleOutIntervalSeconds = nil
	tac.Spec.TiDB.ScaleInIntervalSeconds = nil
	defaultTAC(tac)
	g.Expect(*tac.Spec.TiDB.MinReplicas).Should(Equal(int32(1)))
	g.Expect(len(tac.Spec.TiDB.Metrics)).Should(Equal(1))
	g.Expect(*tac.Spec.TiDB.MetricsTimeDuration).Should(Equal("3m"))
	g.Expect(*tac.Spec.TiDB.ScaleOutIntervalSeconds).Should(Equal(int32(300)))
	g.Expect(*tac.Spec.TiDB.ScaleInIntervalSeconds).Should(Equal(int32(500)))

}

func TestCheckAndUpdateTacAnn(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name            string
		haveScaling     bool
		targetNamespace string
		targetName      string
		markedNamespace string
		markedName      string
	}{
		{
			name:            "first syncing",
			haveScaling:     false,
			markedName:      "",
			markedNamespace: "",
			targetName:      "foo",
			targetNamespace: "bar",
		},
		{
			name:            "second syncing",
			haveScaling:     true,
			markedName:      "foo",
			markedNamespace: "bar",
			targetName:      "foo",
			targetNamespace: "bar",
		},
		{
			name:            "change target",
			haveScaling:     true,
			markedName:      "foo",
			markedNamespace: "bar",
			targetName:      "foo2",
			targetNamespace: "bar2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tac := newTidbClusterAutoScaler()
			tac.Annotations = nil
			tac.Spec.Cluster.Name = tt.targetName
			tac.Spec.Cluster.Namespace = tt.targetNamespace
			if tt.haveScaling {
				tac.Annotations = map[string]string{}
				tac.Annotations[label.AnnAutoScalingTargetNamespace] = tt.targetNamespace
				tac.Annotations[label.AnnAutoScalingTargetName] = tt.targetName
			}
			checkAndUpdateTacAnn(tac)
			v, ok := tac.Annotations[label.AnnAutoScalingTargetNamespace]
			g.Expect(ok).Should(Equal(ok))
			g.Expect(v).Should(Equal(tt.targetNamespace))
			v, ok = tac.Annotations[label.AnnAutoScalingTargetName]
			g.Expect(ok).Should(Equal(ok))
			g.Expect(v).Should(Equal(tt.targetName))
		})
	}
}

func TestGenMetricsEndpoint(t *testing.T) {
	g := NewGomegaWithT(t)
	tac := newTidbClusterAutoScaler()
	tac.Spec.Monitor = nil
	r, err := genMetricsEndpoint(tac)
	g.Expect(err).ShouldNot(BeNil())
	g.Expect(err.Error()).Should(Equal(fmt.Sprintf("tac[%s/%s] metrics url or monitor should be defined explicitly", tac.Namespace, tac.Name)))
	g.Expect(r).Should(Equal(""))

	tac.Spec.Monitor = &v1alpha1.TidbMonitorRef{
		Name:      "monitor",
		Namespace: "default",
	}
	r, err = genMetricsEndpoint(tac)
	g.Expect(err).Should(BeNil())
	g.Expect(r).Should(Equal(fmt.Sprintf("http://%s-prometheus.%s.svc:9090", tac.Spec.Monitor.Name, tac.Spec.Monitor.Namespace)))

	u := "metrics-url"
	tac.Spec.MetricsUrl = &u
	r, err = genMetricsEndpoint(tac)
	g.Expect(err).Should(BeNil())
	g.Expect(r).Should(Equal(u))
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
	tac.Spec.Monitor = &v1alpha1.TidbMonitorRef{
		Namespace: "monitor",
		Name:      "default",
	}
	tac.Spec.TiKV = &v1alpha1.TikvAutoScalerSpec{}
	tac.Spec.TiDB = &v1alpha1.TidbAutoScalerSpec{}
	tac.Spec.TiKV.ScaleOutThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiKV.ScaleInThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiDB.ScaleOutThreshold = pointer.Int32Ptr(2)
	tac.Spec.TiDB.ScaleInThreshold = pointer.Int32Ptr(2)
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
