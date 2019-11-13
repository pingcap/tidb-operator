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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
		name         string
		memberType   v1alpha1.MemberType
		ordinal      int32
		pvc          *corev1.PersistentVolumeClaim
		deleteFailed bool
		expectFn     func(*GomegaWithT, map[int32]string, error)
	}
	tc := newTidbClusterForPD()
	setName := controller.PDMemberName(tc.GetName())
	testFn := func(test *testcase, t *testing.T) {
		t.Logf(test.name)
		g := NewGomegaWithT(t)

		gs, pvcIndexer, pvcControl := newFakeGeneralScaler()
		if test.pvc != nil {
			pvcIndexer.Add(test.pvc)
		}
		if test.deleteFailed {
			pvcControl.SetDeletePVCError(fmt.Errorf("delete pvc failed"), 0)
		}

		skipReason, err := gs.deleteDeferDeletingPVC(tc, setName, test.memberType, test.ordinal)
		test.expectFn(g, skipReason, err)
	}
	tests := []testcase{
		{
			name:       "normal",
			memberType: v1alpha1.PDMemberType,
			ordinal:    3,
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
					Namespace: corev1.NamespaceDefault,
					Annotations: map[string]string{
						label.AnnPVCDeferDeleting: "deleting-3",
					},
				},
			},
			deleteFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
		{
			name:         "pvc is not found",
			memberType:   v1alpha1.PDMemberType,
			ordinal:      3,
			pvc:          nil,
			deleteFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[3]).To(Equal(skipReasonScalerPVCNotFound))
			},
		},
		{
			name:       "pvc annotations is nil",
			memberType: v1alpha1.PDMemberType,
			ordinal:    3,
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
					Namespace: corev1.NamespaceDefault,
				},
			},
			deleteFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[3]).To(Equal(skipReasonScalerAnnIsNil))
			},
		},
		{
			name:       "pvc annotations defer deleting is empty",
			memberType: v1alpha1.PDMemberType,
			ordinal:    3,
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
					Namespace:   corev1.NamespaceDefault,
					Annotations: map[string]string{},
				},
			},
			deleteFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[int32]string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason[3]).To(Equal(skipReasonScalerAnnDeferDeletingIsEmpty))
			},
		},
		{
			name:       "pvc delete failed",
			memberType: v1alpha1.PDMemberType,
			ordinal:    3,
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 3),
					Namespace: corev1.NamespaceDefault,
					Annotations: map[string]string{
						label.AnnPVCDeferDeleting: "deleting-3",
					},
				},
			},
			deleteFailed: true,
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
