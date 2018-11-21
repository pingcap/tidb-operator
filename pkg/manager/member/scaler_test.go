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
		name         string
		memberType   v1alpha1.MemberType
		from         int32
		to           int32
		pvc          *corev1.PersistentVolumeClaim
		deletePVCErr bool
		expectFn     func(*GomegaWithT, error)
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
		if test.deletePVCErr {
			pvcControl.SetDeletePVCError(fmt.Errorf("delete pvc failed"), 0)
		}

		err := gs.deleteAllDeferDeletingPVC(tc, setName, test.memberType, test.from, test.to)
		test.expectFn(g, err)
	}
	tests := []testcase{
		{
			name:         "from 4 to 5, but pvc is not found",
			memberType:   v1alpha1.PDMemberType,
			from:         4,
			to:           5,
			pvc:          nil,
			deletePVCErr: false,
			expectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:       "annotation is nil",
			memberType: v1alpha1.PDMemberType,
			from:       4,
			to:         5,
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
					Namespace: corev1.NamespaceDefault,
				},
			},
			deletePVCErr: false,
			expectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:       "annotation defer deleting is empty",
			memberType: v1alpha1.PDMemberType,
			from:       4,
			to:         5,
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
					Namespace:   corev1.NamespaceDefault,
					Annotations: map[string]string{},
				},
			},
			deletePVCErr: false,
			expectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:       "delete pvc failed",
			memberType: v1alpha1.PDMemberType,
			from:       4,
			to:         5,
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ordinalPVCName(v1alpha1.PDMemberType, setName, 4),
					Namespace: corev1.NamespaceDefault,
					Annotations: map[string]string{
						label.AnnPVCDeferDeleting: "xxx",
					},
				},
			},
			deletePVCErr: true,
			expectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "delete pvc failed")).To(BeTrue())
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
