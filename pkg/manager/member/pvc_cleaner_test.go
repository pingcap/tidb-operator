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

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestPVCCleanerClean(t *testing.T) {
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

		pcc, podIndexer, pvcIndexer, pvcControl := newFakePVCCleaner()
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

		skipReason, err := pcc.Clean(tc)
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).TiDB().Labels(),
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["tidb-test-tidb-0"]).To(Equal(skipReasonPVCCleanerIsNotPDOrTiKV))
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
						Labels:      label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
				pvc, err := pcc.pvcLister.PersistentVolumeClaims(metav1.NamespaceDefault).Get("pd-test-pd-0")
				g.Expect(err).NotTo(HaveOccurred())
				_, exist := pvc.Labels[label.AnnPVCPodScheduling]
				g.Expect(exist).To(BeFalse())
			},
		},
		{
			name: "pvc's meta info has not to be synced",
			pods: nil,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
						Annotations: map[string]string{
							label.AnnPVCPodScheduling: "true",
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *realPVCCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pd-test-pd-0"]).To(Equal(skipReasonPVCCleanerWaitingForPVCSync))
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: metav1.NamespaceDefault,
						Name:      "pd-test-pd-0",
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
						Labels:    label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).PD().Labels(),
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
				pvc, err := pcc.pvcLister.PersistentVolumeClaims(metav1.NamespaceDefault).Get("pd-test-pd-0")
				g.Expect(err).NotTo(HaveOccurred())
				_, exist := pvc.Labels[label.AnnPVCPodScheduling]
				g.Expect(exist).To(BeFalse())
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakePVCCleaner() (*realPVCCleaner, cache.Indexer, cache.Indexer, *controller.FakePVCControl) {
	kubeCli := kubefake.NewSimpleClientset()
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	podInformer := kubeInformerFactory.Core().V1().Pods()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvcControl := controller.NewFakePVCControl(pvcInformer)

	return &realPVCCleaner{podInformer.Lister(), pvcControl, pvcInformer.Lister()},
		podInformer.Informer().GetIndexer(), pvcInformer.Informer().GetIndexer(), pvcControl
}
