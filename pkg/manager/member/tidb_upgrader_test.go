// Copyright 2018 PingCAP, Inc.
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

package member

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestTiDBUpgrader_Upgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                    string
		changeFn                func(*v1alpha1.TidbCluster)
		getLastAppliedConfigErr bool
		resignDDLOwnerError     bool
		errorExpect             bool
		expectFn                func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		upgrader, tidbControl := newTiDBUpgrader()
		if test.resignDDLOwnerError {
			tidbControl.SetResignDDLOwnerError(fmt.Errorf("resign DDL owner failed"))
		}
		tc := newTidbClusterForTiDBUpgrader()
		if test.changeFn != nil {
			test.changeFn(tc)
		}
		oldSet := newStatefulSetForTiDBUpgrader()
		newSet := oldSet.DeepCopy()
		if test.getLastAppliedConfigErr {
			oldSet.SetAnnotations(map[string]string{LastAppliedConfigAnnotation: "fake apply config"})
		} else {
			SetLastAppliedConfigAnnotation(oldSet)
		}
		newSet.Spec.Template.Spec.Containers[0].Image = "tidb-test-images:v2"

		err := upgrader.Upgrade(tc, oldSet, newSet)
		if test.errorExpect {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		test.expectFn(g, tc, newSet)
	}

	tests := []*testcase{
		{
			name: "normal",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal((func() *int32 { i := int32(0); return &i }())))
			},
		},
		{
			name: "pd is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.UpgradePhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal((func() *int32 { i := int32(1); return &i }())))
			},
		},
		{
			name: "tikv is upgrading",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
			},
			getLastAppliedConfigErr: false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal((func() *int32 { i := int32(1); return &i }())))
			},
		},
		{
			name: "get apply config error",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
			},
			getLastAppliedConfigErr: true,
			errorExpect:             true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal((func() *int32 { i := int32(1); return &i }())))
			},
		},
		{
			name: "upgraded pods are not ready",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.Members["upgrader-tidb-1"] = v1alpha1.TiDBMember{
					Name:   "upgrader-tidb-1",
					Health: false,
				}
			},
			getLastAppliedConfigErr: false,
			errorExpect:             true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal((func() *int32 { i := int32(1); return &i }())))
			},
		},
		{
			name: "resign DDL owner error",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
			},
			getLastAppliedConfigErr: false,
			resignDDLOwnerError:     true,
			errorExpect:             true,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal((func() *int32 { i := int32(1); return &i }())))
				g.Expect(tc.Status.TiDB.ResignDDLOwnerFailCount).To(Equal(int32(1)))
			},
		},
		{
			name: "resign DDL owner error count larger than MaxResignDDLOwnerCount",
			changeFn: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Phase = v1alpha1.NormalPhase
				tc.Status.TiKV.Phase = v1alpha1.NormalPhase
				tc.Status.TiDB.ResignDDLOwnerFailCount = 4
			},
			getLastAppliedConfigErr: false,
			resignDDLOwnerError:     true,
			errorExpect:             false,
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet) {
				g.Expect(newSet.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal((func() *int32 { i := int32(0); return &i }())))
				g.Expect(tc.Status.TiDB.ResignDDLOwnerFailCount).To(Equal(int32(0)))
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}

}

func newTiDBUpgrader() (Upgrader, *controller.FakeTiDBControl) {
	tidbControl := controller.NewFakeTiDBControl()
	return &tidbUpgrader{tidbControl: tidbControl}, tidbControl
}

func newStatefulSetForTiDBUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader-tidb",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: int32Pointer(2),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tidb",
							Image: "tidb-test-image",
						},
					},
				},
			},
			UpdateStrategy: apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
					Partition: int32Pointer(1),
				},
			},
		},
		Status: apps.StatefulSetStatus{
			CurrentRevision: "1",
			UpdateRevision:  "2",
			ReadyReplicas:   2,
			Replicas:        2,
			CurrentReplicas: 1,
			UpdatedReplicas: 1,
		},
	}
}

func newTidbClusterForTiDBUpgrader() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("upgrader"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: v1alpha1.PDSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "pd-test-image",
				},
				Replicas:         3,
				StorageClassName: "my-storage-class",
			},
			TiKV: v1alpha1.TiKVSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "tikv-test-image",
				},
				Replicas:         3,
				StorageClassName: "my-storage-class",
			},
			TiDB: v1alpha1.TiDBSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "tidb-test-image",
				},
				Replicas:         2,
				StorageClassName: "my-storage-class",
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			TiDB: v1alpha1.TiDBStatus{
				StatefulSet: &apps.StatefulSetStatus{
					CurrentReplicas: 1,
					UpdatedReplicas: 1,
					Replicas:        2,
				},
				Members: map[string]v1alpha1.TiDBMember{
					"upgrader-tidb-0": {
						Name:   "upgrader-tidb-0",
						Health: true,
					},
					"upgrader-tidb-1": {
						Name:   "upgrader-tidb-1",
						Health: true,
					},
				},
			},
		},
	}
}
