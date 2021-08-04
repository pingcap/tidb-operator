// Copyright 2021 PingCAP, Inc.
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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
)

func TestTiDBScalerScaleOut(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name          string
		tidbUpgrading bool
		hasPVC        bool
		hasDeferAnn   bool
		pvcDeleteErr  bool
		annoIsNil     bool
		errExpectFn   func(*GomegaWithT, error)
		changed       bool
	}

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()

		if test.tidbUpgrading {
			tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		oldSet.Name = fmt.Sprintf("%s-tidb", tc.Name)
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(7)

		scaler, pvcIndexer, _, pvcControl := newFakeTiDBScaler()

		pvc := newPVCForStatefulSet(oldSet, v1alpha1.TiDBMemberType, tc.Name)
		pvc.Name = ordinalPVCName(v1alpha1.TiDBMemberType, fmt.Sprintf("log-%s", oldSet.Name), *oldSet.Spec.Replicas)
		if !test.annoIsNil {
			pvc.Annotations = map[string]string{}
		}

		if test.hasDeferAnn {
			pvc.Annotations = map[string]string{}
			pvc.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)
		}
		if test.hasPVC {
			pvcIndexer.Add(pvc)
		}

		if test.pvcDeleteErr {
			pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := scaler.ScaleOut(tc, oldSet, newSet)
		test.errExpectFn(g, err)
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(6))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:          "normal",
			tidbUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   false,
			annoIsNil:     true,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "tidb is upgrading",
			tidbUpgrading: true,
			hasPVC:        true,
			hasDeferAnn:   false,
			annoIsNil:     true,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "cache don't have pvc",
			tidbUpgrading: false,
			hasPVC:        false,
			hasDeferAnn:   false,
			annoIsNil:     true,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "pvc annotation is not nil but doesn't contain defer deletion annotation",
			tidbUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   false,
			annoIsNil:     false,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "pvc annotations defer deletion is not nil, pvc delete failed",
			tidbUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   true,
			pvcDeleteErr:  true,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestTiDBScalerScaleIn(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name          string
		tidbUpgrading bool
		hasPVC        bool
		isPodReady    bool
		hasSynced     bool
		pvcUpdateErr  bool
		errExpectFn   func(*GomegaWithT, error)
		changed       bool
	}

	resyncDuration := time.Duration(0)

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()

		if test.tidbUpgrading {
			tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(3)

		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:              tidbPodName(tc.GetName(), 4),
				Namespace:         corev1.NamespaceDefault,
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
		}

		readyPodFunc(pod)
		if !test.isPodReady {
			notReadyPodFunc(pod)
		}

		if !test.hasSynced {
			pod.CreationTimestamp = metav1.Time{Time: time.Now().Add(1 * time.Hour)}
		}

		scaler, pvcIndexer, podIndexer, pvcControl := newFakeTiDBScaler(resyncDuration)

		if test.hasPVC {
			pvc1 := newScaleInPVCForStatefulSet(oldSet, v1alpha1.TiDBMemberType, tc.Name)
			pvc1.Name = ordinalPVCName(v1alpha1.TiDBMemberType, fmt.Sprintf("log-%s", oldSet.Name), *oldSet.Spec.Replicas-1)
			pvc2 := pvc1.DeepCopy()
			pvc1.Name = pvc1.Name + "-1"
			pvc1.UID = pvc1.UID + "-1"
			pvc2.Name = pvc2.Name + "-2"
			pvc2.UID = pvc2.UID + "-2"
			pvcIndexer.Add(pvc1)
			pvcIndexer.Add(pvc2)
			pod.Spec.Volumes = append(pod.Spec.Volumes,
				corev1.Volume{
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc1.Name,
						},
					},
				}, corev1.Volume{
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc2.Name,
						},
					},
				})
		}

		pod.Labels = map[string]string{}
		podIndexer.Add(pod)

		if test.pvcUpdateErr {
			pvcControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := scaler.ScaleIn(tc, oldSet, newSet)
		test.errExpectFn(g, err)
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(4))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:          "able to scale in while not upgrading",
			tidbUpgrading: false,
			hasPVC:        true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "able to scale in while upgrading",
			tidbUpgrading: true,
			hasPVC:        true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "tidb pod is not ready now, not sure if the status has been synced",
			tidbUpgrading: false,
			hasPVC:        true,
			isPodReady:    false,
			hasSynced:     false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "tidb pod is not ready now, make sure the status has been synced",
			tidbUpgrading: false,
			hasPVC:        true,
			isPodReady:    false,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "tidb pod is ready now, but the status has not been synced",
			tidbUpgrading: false,
			hasPVC:        true,
			isPodReady:    true,
			hasSynced:     false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "don't have pvc",
			tidbUpgrading: false,
			hasPVC:        false,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "update PVC failed",
			tidbUpgrading: false,
			hasPVC:        true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  true,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func newFakeTiDBScaler(resyncDuration ...time.Duration) (*tidbScaler, cache.Indexer, cache.Indexer, *controller.FakePVCControl) {
	fakeDeps := controller.NewFakeDependencies()
	if len(resyncDuration) > 0 {
		fakeDeps.CLIConfig.ResyncDuration = resyncDuration[0]
	}
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	pvcControl := fakeDeps.PVCControl.(*controller.FakePVCControl)
	return &tidbScaler{generalScaler{deps: fakeDeps}}, pvcIndexer, podIndexer, pvcControl
}
