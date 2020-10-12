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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestPVCCleanerReclaimPV(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbClusterForPD()
	type testcase struct {
		name             string
		pvReclaimEnabled bool
		pods             []*corev1.Pod
		apiPods          []*corev1.Pod
		pvcs             []*corev1.PersistentVolumeClaim
		apiPvcs          []*corev1.PersistentVolumeClaim
		pvs              []*corev1.PersistentVolume
		getPodFailed     bool
		patchPVFailed    bool
		getPVCFailed     bool
		deletePVCFailed  bool
		expectFn         func(*GomegaWithT, map[string]string, *realPVCCleaner, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		tc.Spec.EnablePVReclaim = pointer.BoolPtr(test.pvReclaimEnabled)
		pcc, fakeCli, podIndexer, pvcIndexer, pvcControl, pvIndexer, pvControl := newFakePVCCleaner()
		if test.pods != nil {
			for _, pod := range test.pods {
				podIndexer.Add(pod)
			}
		}
		if test.apiPods != nil {
			for _, apiPod := range test.apiPods {
				fakeCli.CoreV1().Pods(apiPod.GetNamespace()).Create(apiPod)
			}
		}
		if test.pvcs != nil {
			for _, pvc := range test.pvcs {
				pvcIndexer.Add(pvc)
			}
		}
		if test.apiPvcs != nil {
			for _, apiPvc := range test.apiPvcs {
				fakeCli.CoreV1().PersistentVolumeClaims(apiPvc.GetNamespace()).Create(apiPvc)
			}
		}
		if test.pvs != nil {
			for _, pv := range test.pvs {
				pvIndexer.Add(pv)
			}
		}
		if test.getPodFailed {
			fakeCli.PrependReactor("get", "pods", func(action core.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("API server down")
			})
		}
		if test.patchPVFailed {
			pvControl.SetUpdatePVError(fmt.Errorf("patch pv failed"), 0)
		}
		if test.getPVCFailed {
			fakeCli.PrependReactor("get", "persistentvolumeclaims", func(action core.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("API server down")
			})
		}
		if test.deletePVCFailed {
			pvcControl.SetDeletePVCError(fmt.Errorf("delete pvc failed"), 0)
		}

		skipReason, err := pcc.reclaimPV(tc)
		test.expectFn(g, skipReason, pcc, err)
	}
	tests := []testcase{
		{
			name:             "no pvcs",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs:             nil,
			apiPvcs:          nil,
			pvs:              nil,
			getPodFailed:     false,
			patchPVFailed:    false,
			getPVCFailed:     false,
			deletePVCFailed:  false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
		{
			name:             "not pd or tikv pvcs",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "tidb-test-tidb-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).TiDB().Labels(),
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["tidb-test-tidb-0"]).To(Equal(skipReasonPVCCleanerIsNotTarget))
			},
		},
		{
			name:             "pv reclaim is not enabled",
			pvReclaimEnabled: false,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "tidb-test-tidb-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
		{
			name:             "pvc is not bound",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "tidb-test-tidb-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimPending,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["tidb-test-tidb-0"]).To(Equal(skipReasonPVCCleanerPVCNotBound))
			},
		},
		{
			name:             "pvc has been deleted",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         metav1.NamespaceDefault,
						Name:              "tidb-test-tidb-0",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Labels:            label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["tidb-test-tidb-0"]).To(Equal(skipReasonPVCCleanerPVCHasBeenDeleted))
			},
		},
		{
			name:             "pvc is not defer delete pvc",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerIsNotDeferDeletePVC))
			},
		},
		{
			name:             "pvc not has pod name annotation",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPVCNotHasPodNameAnn))
			},
		},
		{
			name:             "pvc is still referenced by pod,",
			pvReclaimEnabled: true,
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pd-0",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
				},
			},
			apiPods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPVCeferencedByPod))
			},
		},
		{
			name:             "not found pod that referenced pvc from local cache, but found this pod from apiserver",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pd-0",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPVCeferencedByPod))
			},
		},
		{
			name:             "get pod from apiserver failed",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pd-0",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    true,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(strings.Contains(err.Error(), "API server down")).To(BeTrue())
			},
		},
		{
			name:             "not found pv bound to pvc",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs:         nil,
			pvs:             nil,
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerNotFoundPV))
			},
		},
		{
			name:             "patch pv reclaim policy to delete policy failed",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs: nil,
			pvs: []*corev1.PersistentVolume{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pd-local-pv-0",
						Namespace: metav1.NamespaceAll,
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					},
				},
			},
			getPodFailed:    false,
			patchPVFailed:   true,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(strings.Contains(err.Error(), "patch pv failed")).To(BeTrue())
			},
		},
		{
			name:             "found pvc from local cache, but not found this pvc from apiserver",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs: nil,
			pvs: []*corev1.PersistentVolume{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pd-local-pv-0",
						Namespace: metav1.NamespaceAll,
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					},
				},
			},
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPVCNotFound))
			},
		},
		{
			name:             "get pvc from apiserver failed",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "1",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "1",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			pvs: []*corev1.PersistentVolume{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pd-local-pv-0",
						Namespace: metav1.NamespaceAll,
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					},
				},
			},
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    true,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(strings.Contains(err.Error(), "API server down")).To(BeTrue())
			},
		},
		{
			name:             "pvc has been changed",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "1",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "2",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			pvs: []*corev1.PersistentVolume{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pd-local-pv-0",
						Namespace: metav1.NamespaceAll,
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					},
				},
			},
			getPodFailed:    false,
			patchPVFailed:   false,
			getPVCFailed:    false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPVCChanged))
			},
		},
		{
			name:             "delete pvc failed",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "1",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "1",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			pvs: []*corev1.PersistentVolume{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pd-local-pv-0",
						Namespace: metav1.NamespaceAll,
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					},
				},
			},
			getPodFailed:    false,
			patchPVFailed:   false,
			deletePVCFailed: true,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, pcc *realPVCCleaner, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
				pv, pvGetErr := pcc.deps.PVLister.Get("pd-local-pv-0")
				g.Expect(pvGetErr).NotTo(HaveOccurred())
				g.Expect(pv.Spec.PersistentVolumeReclaimPolicy).To(Equal(corev1.PersistentVolumeReclaimDelete))
				g.Expect(strings.Contains(err.Error(), "delete pvc failed")).To(BeTrue())
			},
		},
		{
			name:             "the defer delete pvc has been remove and reclaim pv success",
			pvReclaimEnabled: true,
			pods:             nil,
			apiPods:          nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "1",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			apiPvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       metav1.NamespaceDefault,
						Name:            "pd-test-pd-0",
						UID:             types.UID("pd-test"),
						ResourceVersion: "1",
						Labels:          label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: time.Now().Format(time.RFC3339),
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "pd-local-pv-0",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimBound,
					},
				},
			},
			pvs: []*corev1.PersistentVolume{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pd-local-pv-0",
						Namespace: metav1.NamespaceAll,
					},
					Spec: corev1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					},
				},
			},
			getPodFailed:    false,
			patchPVFailed:   false,
			deletePVCFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, pcc *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
				pv, pvGetErr := pcc.deps.PVLister.Get("pd-local-pv-0")
				g.Expect(pvGetErr).NotTo(HaveOccurred())
				_, pvcGetErr := pcc.deps.PVCLister.PersistentVolumeClaims(metav1.NamespaceAll).Get("pd-test-pd-0")
				g.Expect(errors.IsNotFound(pvcGetErr)).To(BeTrue())
				g.Expect(pv.Spec.PersistentVolumeReclaimPolicy).To(Equal(corev1.PersistentVolumeReclaimDelete))
			},
		},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			testFn(&tests[i], t)
		})
	}
}

func TestPVCCleanerCleanScheduleLock(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbClusterForPD()
	type testcase struct {
		name            string
		pods            []*corev1.Pod
		pvcs            []*corev1.PersistentVolumeClaim
		updatePVCFailed bool
		expectFn        func(*GomegaWithT, map[string]string, *realPVCCleaner, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		pcc, _, podIndexer, pvcIndexer, pvcControl, _, _ := newFakePVCCleaner()
		if test.pods != nil {
			for _, pod := range test.pods {
				podIndexer.Add(pod)
			}
		}
		if test.pvcs != nil {
			for _, pvc := range test.pvcs {
				pvcIndexer.Add(pvc)
			}
		}
		if test.updatePVCFailed {
			pvcControl.SetUpdatePVCError(fmt.Errorf("update PVC failed"), 0)
		}

		skipReason, err := pcc.cleanScheduleLock(tc)
		test.expectFn(g, skipReason, pcc, err)
	}
	tests := []testcase{
		{
			name: "no pvcs",
			pods: []*corev1.Pod{},
			pvcs: nil,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
		{
			name: "not pd or tikv pvcs",
			pods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "tidb-test-tidb-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).TiDB().Labels(),
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["tidb-test-tidb-0"]).To(Equal(skipReasonPVCCleanerIsNotTarget))
			},
		},
		{
			name: "defer delete pvc that not has the schedule lock",
			pods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   metav1.NamespaceDefault,
						Name:        "pd-test-pd-0",
						Labels:      label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{label.AnnPVCDeferDeleting: "true"},
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerDeferDeletePVCNotHasLock))
			},
		},
		{
			name: "defer delete pvc that has the schedule lock but update pvc failed",
			pods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "true",
							label.AnnPVCPodScheduling: "true",
						},
					},
				},
			},
			updatePVCFailed: true,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update PVC failed")).To(BeTrue())
			},
		},
		{
			name: "defer delete pvc that has the schedule lock and update pvc success",
			pods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "true",
							label.AnnPVCPodScheduling: "true",
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, pcc *realPVCCleaner, err error) {
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(err).NotTo(HaveOccurred())
				pvc, err := pcc.deps.PVCLister.PersistentVolumeClaims(metav1.NamespaceDefault).Get("pd-test-pd-0")
				g.Expect(err).NotTo(HaveOccurred())
				_, exist := pvc.Labels[label.AnnPVCPodScheduling]
				g.Expect(exist).To(BeFalse())
			},
		},
		{
			name: "pvc not has pod name annotation",
			pods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCPodScheduling: "true",
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPVCNotHasPodNameAnn))
			},
		},
		{
			name: "the corresponding pod of pvc has not been found",
			pods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCPodScheduling: "true",
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPodNotFound))
			},
		},
		{
			name: "pvc that not has the schedule lock",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pd-0",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPodNameKey: "test-pd-0",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPVCNotHasLock))
			},
		},
		{
			name: "waiting for pod scheduling",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pd-0",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCPodScheduling: "true",
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerPodWaitingForScheduling))
			},
		},
		{
			name: "pvc that need to remove the schedule lock but update pvc failed",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pd-0",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						NodeName: "node-1",
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCPodScheduling: "true",
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
				},
			},
			updatePVCFailed: true,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update PVC failed")).To(BeTrue())
			},
		},
		{
			name: "pvc that need to remove the schedule lock and update pvc success",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pd-0",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						NodeName: "node-1",
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCPodScheduling: "true",
							label.AnnPodNameKey:       "test-pd-0",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, pcc *realPVCCleaner, err error) {
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(err).NotTo(HaveOccurred())
				pvc, err := pcc.deps.PVCLister.PersistentVolumeClaims(metav1.NamespaceDefault).Get("pd-test-pd-0")
				g.Expect(err).NotTo(HaveOccurred())
				_, exist := pvc.Labels[label.AnnPVCPodScheduling]
				g.Expect(exist).To(BeFalse())
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			testFn(&tests[i], t)
		})
	}
}

func newFakePVCCleaner() (*realPVCCleaner, *kubefake.Clientset, cache.Indexer, cache.Indexer, *controller.FakePVCControl, cache.Indexer, *controller.FakePVControl) {
	fakeDeps := controller.NewFakeDependencies()
	rpc := &realPVCCleaner{deps: fakeDeps}
	kubeCli := fakeDeps.KubeClientset.(*kubefake.Clientset)
	podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	pvcControl := fakeDeps.PVCControl.(*controller.FakePVCControl)
	pvIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumes().Informer().GetIndexer()
	pvControl := fakeDeps.PVControl.(*controller.FakePVControl)
	return rpc, kubeCli, podIndexer, pvcIndexer, pvcControl, pvIndexer, pvControl
}
