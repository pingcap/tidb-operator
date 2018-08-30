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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestTiKVUpgrader_Upgrade(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name        string
		pdUpgrading bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		upgrader := newTiKVUpgrader()
		tc := newTidbClusterForTiKVUpgrader()
		if test.pdUpgrading {
			tc.Status.PD.Phase = v1alpha1.UpgradePhase
		} else {
			tc.Status.PD.Phase = v1alpha1.NormalPhase
		}
		oldSet := newStatefulSetForTiKVUpgrader()
		newSet := oldSet.DeepCopy()
		SetLastAppliedConfigAnnotation(oldSet)
		newSet.Spec.Template.Spec.Containers[0].Image = "tikv-test-images:v2"

		err := upgrader.Upgrade(tc, oldSet, newSet)
		g.Expect(err).NotTo(HaveOccurred())
		if test.pdUpgrading {
			g.Expect(newSet.Spec.Template.Spec).To(Equal(oldSet.Spec.Template.Spec))
			g.Expect(tc.Status.TiKV.Phase).NotTo(Equal(v1alpha1.UpgradePhase))
		} else {
			g.Expect(tc.Status.TiKV.Phase).To(Equal(v1alpha1.UpgradePhase))
		}
	}

	tests := []*testcase{
		{name: "pd is upgrading", pdUpgrading: true},
		{name: "normal", pdUpgrading: false},
	}

	for _, test := range tests {
		testFn(test, t)
	}

}

func newTiKVUpgrader() Upgrader {
	return &tikvUpgrader{}
}

func newStatefulSetForTiKVUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader-tikv",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: int32Pointer(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tikv",
							Image: "tikv-test-image",
						},
					},
				},
			},
		},
	}
}

func newTidbClusterForTiKVUpgrader() *v1alpha1.TidbCluster {
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
		},
	}
}
