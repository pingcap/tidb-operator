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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestGeneralScalerDeleteAllDeferDeletingPVC(t *testing.T) {
	type testcase struct {
		name            string
		memberType      v1alpha1.MemberType
		from            int32
		to              int32
		pvcs            []*corev1.PersistentVolumeClaim
		deleteFailedIdx []int
		expectFn        func(*GomegaWithT, map[int32]string, error)
	}
	tc := newTidbClusterForPD()
	setName := controller.PDMemberName(tc.GetName())
	testFn := func(test *testcase, t *testing.T) {
		t.Logf(test.name)
		g := NewGomegaWithT(t)

		gs, pvcIndexer, pvcControl := newFakeGeneralScaler()
		for _, pvc := range test.pvcs {
			pvcIndexer.Add(pvc)
		}
		for _, idx := range test.deleteFailedIdx {
			pvcControl.SetDeletePVCError(fmt.Errorf("delete pvc failed"), idx)
		}

		skipReason, err := gs.deleteAllDeferDeletingPVC(tc, setName, test.memberType, test.from, test.to)
		test.expectFn(g, skipReason, err)
	}
	tests := []testcase{
		{
			name:       "normal",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-3",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-4",
						},
					},
				},
			},
			deleteFailedIdx: nil,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
		{
			name:       "from 3 to 5, but pvc 3 is not found",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-4",
						},
					},
				},
			},
			deleteFailedIdx: nil,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[3]).To(Equal(skipReasonPVCNotFound))
			},
		},
		{
			name:       "from 3 to 5, but pvc 4 is not found",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-3",
						},
					},
				},
			},
			deleteFailedIdx: nil,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[4]).To(Equal(skipReasonPVCNotFound))
			},
		},
		{
			name:       "from 3 to 5, but pvc 3 annotations is nil",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace: corev1.NamespaceDefault,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-4",
						},
					},
				},
			},
			deleteFailedIdx: nil,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[3]).To(Equal(skipReasonAnnIsNil))
			},
		},
		{
			name:       "from 3 to 5, but pvc 4 annotations is nil",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-3",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace: corev1.NamespaceDefault,
					},
				},
			},
			deleteFailedIdx: nil,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[4]).To(Equal(skipReasonAnnIsNil))
			},
		},
		{
			name:       "from 3 to 5, but pvc 3 annotations defer deleting is empty",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace:   corev1.NamespaceDefault,
						Annotations: map[string]string{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-4",
						},
					},
				},
			},
			deleteFailedIdx: nil,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[3]).To(Equal(skipReasonAnnDeferDeletingIsEmpty))
			},
		},
		{
			name:       "from 3 to 5, but pvc 4 annotations defer deleting is empty",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-3",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace:   corev1.NamespaceDefault,
						Annotations: map[string]string{},
					},
				},
			},
			deleteFailedIdx: nil,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[4]).To(Equal(skipReasonAnnDeferDeletingIsEmpty))
			},
		},
		{
			name:       "from 3 to 5, but pvc 3 delete failed",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-3",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-4",
						},
					},
				},
			},
			deleteFailedIdx: []int{0},
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "delete pvc failed")).To(BeTrue())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
		{
			name:       "from 3 to 5, but pvc 4 delete failed",
			memberType: v1alpha1.PDMemberType,
			from:       3,
			to:         5,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-3",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
						Namespace: corev1.NamespaceDefault,
						Annotations: map[string]string{
							label.AnnPVCDeferDeleting: "deleting-4",
						},
					},
				},
			},
			deleteFailedIdx: []int{1},
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "delete pvc failed")).To(BeTrue())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeGeneralScaler() (*generalScaler, cache.Indexer, *controller.FakePVCControl) {
	kubeCli := kubefake.NewSimpleClientset()
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvcControl := controller.NewFakePVCControl(pvcInformer)

	return &generalScaler{pvcLister: pvcInformer.Lister(), pvcControl: pvcControl},
		pvcInformer.Informer().GetIndexer(), pvcControl
}
