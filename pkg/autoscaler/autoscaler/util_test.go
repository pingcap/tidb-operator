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
	tac := newTidbClusterAutoScaler()
	intervalSec := int32(100)
	r, err := checkStsAutoScalingInterval(tac, intervalSec, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(true))
	g.Expect(err).Should(BeNil())

	r, err = checkStsAutoScalingInterval(tac, intervalSec, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(true))
	g.Expect(err).Should(BeNil())

	tac.Annotations = map[string]string{}
	tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Truncate(60*time.Second).Unix())
	tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Truncate(60*time.Second).Unix())
	r, err = checkStsAutoScalingInterval(tac, intervalSec, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(false))
	g.Expect(err).Should(BeNil())
	r, err = checkStsAutoScalingInterval(tac, intervalSec, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(false))
	g.Expect(err).Should(BeNil())

	tac.Annotations = map[string]string{}
	tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Truncate(120*time.Second).Unix())
	tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Truncate(120*time.Second).Unix())
	r, err = checkStsAutoScalingInterval(tac, intervalSec, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(true))
	g.Expect(err).Should(BeNil())
	r, err = checkStsAutoScalingInterval(tac, intervalSec, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(true))
	g.Expect(err).Should(BeNil())
}

func TestCheckStsAutoScalingPrerequisites(t *testing.T) {
	g := NewGomegaWithT(t)
	sts := newSts()

	r := checkStsAutoScalingPrerequisites(sts)
	g.Expect(r).Should(Equal(false))

	sts.Status.UpdateRevision = "1"
	sts.Status.CurrentRevision = "1"
	sts.Spec.Replicas = pointer.Int32Ptr(1)
	sts.Status.Replicas = 1
	r = checkStsAutoScalingPrerequisites(sts)
	g.Expect(r).Should(Equal(true))

	sts.Status.UpdateRevision = "1"
	sts.Status.CurrentRevision = "2"
	sts.Spec.Replicas = pointer.Int32Ptr(1)
	sts.Status.Replicas = 1
	r = checkStsAutoScalingPrerequisites(sts)
	g.Expect(r).Should(Equal(false))

	sts.Status.UpdateRevision = "1"
	sts.Status.CurrentRevision = "1"
	sts.Spec.Replicas = pointer.Int32Ptr(1)
	sts.Status.Replicas = 2
	r = checkStsAutoScalingPrerequisites(sts)
	g.Expect(r).Should(Equal(false))
}

func TestLimitTargetReplicas(t *testing.T) {
	g := NewGomegaWithT(t)
	tac := newTidbClusterAutoScaler()
	tac.Spec.TiDB.MinReplicas = pointer.Int32Ptr(2)
	tac.Spec.TiDB.MaxReplicas = 4

	tac.Spec.TiKV.MinReplicas = pointer.Int32Ptr(2)
	tac.Spec.TiKV.MaxReplicas = 4

	targetReplicas := int32(1)
	r := limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(int32(2)))
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(int32(2)))

	targetReplicas = int32(2)
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(int32(2)))
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(int32(2)))

	targetReplicas = int32(3)
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(int32(3)))
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(int32(3)))

	targetReplicas = int32(4)
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(int32(4)))
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(int32(4)))

	targetReplicas = int32(5)
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	g.Expect(r).Should(Equal(int32(4)))
	r = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	g.Expect(r).Should(Equal(int32(4)))
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
	tac := newTidbClusterAutoScaler()

	tac.Annotations = nil
	checkAndUpdateTacAnn(tac)
	g.Expect(tac.Annotations).ShouldNot(BeNil())
	g.Expect(len(tac.Annotations)).Should(Equal(2))
	v, ok := tac.Annotations[label.AnnAutoScalingTargetNamespace]
	g.Expect(ok).Should(Equal(ok))
	g.Expect(v).Should(Equal("default"))
	v, ok = tac.Annotations[label.AnnAutoScalingTargetName]
	g.Expect(ok).Should(Equal(ok))
	g.Expect(v).Should(Equal("tc"))

	tac.Spec.Cluster = v1alpha1.TidbClusterRef{
		Name:      "foo",
		Namespace: "bar",
	}
	checkAndUpdateTacAnn(tac)
	g.Expect(tac.Annotations).ShouldNot(BeNil())
	g.Expect(len(tac.Annotations)).Should(Equal(2))
	v, ok = tac.Annotations[label.AnnAutoScalingTargetNamespace]
	g.Expect(ok).Should(Equal(ok))
	g.Expect(v).Should(Equal("bar"))
	v, ok = tac.Annotations[label.AnnAutoScalingTargetName]
	g.Expect(ok).Should(Equal(ok))
	g.Expect(v).Should(Equal("foo"))
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
