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
// See the License for the specific language governing permissions and
// limitations under the License.

package member

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestPDMSScaler(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name        string
		hasSynced   bool
		errExpectFn func(*GomegaWithT, error)
		changed     bool
		scaleIn     bool
	}

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPDMSScaler()

		if test.hasSynced {
			tc.Status.PDMS[tsoService].Synced = true
		} else {
			tc.Status.PDMS[tsoService].Synced = false
		}

		// default replicas is 5
		oldSet := newStatefulSetForPDMSScale()
		oldSet.Name = fmt.Sprintf("%s-pdms-%s", tc.Name, tsoService)
		newSet := oldSet.DeepCopy()

		scaler := newFakePDMSScaler()
		if test.scaleIn {
			newSet.Spec.Replicas = pointer.Int32Ptr(3)
			err := scaler.ScaleIn(tc, oldSet, newSet)
			test.errExpectFn(g, err)
			if test.changed {
				g.Expect(int(*newSet.Spec.Replicas)).To(Equal(4))
			} else {
				g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
			}
		} else {
			newSet.Spec.Replicas = pointer.Int32Ptr(7)
			err := scaler.ScaleOut(tc, oldSet, newSet)
			test.errExpectFn(g, err)
			if test.changed {
				g.Expect(int(*newSet.Spec.Replicas)).To(Equal(6))
			} else {
				g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
			}
		}
	}

	tests := []testcase{
		{
			name:        "scaleIn normal",
			scaleIn:     true,
			hasSynced:   true,
			errExpectFn: errExpectNil,
			changed:     true,
		},
		{
			name:        "scaleIn pdms is upgrading",
			scaleIn:     true,
			hasSynced:   false,
			errExpectFn: errExpectNotNil,
			changed:     false,
		},
		{
			name:        "scaleOut normal",
			scaleIn:     false,
			hasSynced:   true,
			errExpectFn: errExpectNil,
			changed:     true,
		},
		{
			name:        "scaleOut pdms is upgrading",
			scaleIn:     false,
			hasSynced:   false,
			errExpectFn: errExpectNotNil,
			changed:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func newTidbClusterForPDMSScaler() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pingcap/pd:v7.3.0",
				},
				Replicas:         1,
				StorageClassName: pointer.StringPtr("my-storage-class"),
				Mode:             "ms",
			},
			PDMS: []*v1alpha1.PDMSSpec{
				{
					Name: tsoService,
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: "pd-test-image",
					},
					Replicas:         3,
					StorageClassName: pointer.StringPtr("my-storage-class"),
				},
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			PDMS: map[string]*v1alpha1.PDMSStatus{
				tsoService: {
					Phase: v1alpha1.NormalPhase,
					StatefulSet: &apps.StatefulSetStatus{
						CurrentRevision: "1",
						UpdateRevision:  "2",
						ReadyReplicas:   3,
						Replicas:        3,
						CurrentReplicas: 2,
						UpdatedReplicas: 1,
					},
				},
			},
		},
	}
}

func newFakePDMSScaler(resyncDuration ...time.Duration) *pdMSScaler {
	fakeDeps := controller.NewFakeDependencies()
	if len(resyncDuration) > 0 {
		fakeDeps.CLIConfig.ResyncDuration = resyncDuration[0]
	}
	return &pdMSScaler{generalScaler{deps: fakeDeps}}
}

func newStatefulSetForPDMSScale() *apps.StatefulSet {
	set := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scaler",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(5),
		},
	}
	return set
}
