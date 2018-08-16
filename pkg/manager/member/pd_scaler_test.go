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
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestPDScalerScaleOut(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name         string
		pdUpgrading  bool
		hasPVC       bool
		hasDeferAnn  bool
		pvcDeleteErr bool
		err          bool
		changed      bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()

		if test.pdUpgrading {
			tc.Status.PD.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = int32Pointer(7)

		scaler, _, pvcIndexer, pvcControl := newFakePDScaler()

		pvc := newPVCForStatefulSet(oldSet)
		pvc.Name = fmt.Sprintf("pd-%s-%d", oldSet.GetName(), int(*oldSet.Spec.Replicas)+1)
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
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(6))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:         "normal",
			pdUpgrading:  false,
			hasPVC:       true,
			hasDeferAnn:  false,
			pvcDeleteErr: false,
			err:          false,
			changed:      true,
		},
		{
			name:         "pd is upgrading",
			pdUpgrading:  true,
			hasPVC:       true,
			hasDeferAnn:  false,
			pvcDeleteErr: false,
			err:          false,
			changed:      false,
		},
		{
			name:         "cache don't have pvc",
			pdUpgrading:  false,
			hasPVC:       false,
			hasDeferAnn:  false,
			pvcDeleteErr: false,
			err:          false,
			changed:      true,
		},
		{
			name:         "pvc annotations defer deletion is not nil, pvc delete failed",
			pdUpgrading:  false,
			hasPVC:       true,
			hasDeferAnn:  true,
			pvcDeleteErr: true,
			err:          true,
			changed:      false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDScalerScaleIn(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		pdUpgrading     bool
		hasPVC          bool
		pvcUpdateErr    bool
		deleteMemberErr bool
		err             bool
		changed         bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()

		if test.pdUpgrading {
			tc.Status.PD.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = int32Pointer(3)

		scaler, pdControl, pvcIndexer, pvcControl := newFakePDScaler()

		if test.hasPVC {
			pvc := newPVCForStatefulSet(oldSet)
			pvcIndexer.Add(pvc)
		}

		pdClient := controller.NewFakePDClient()
		pdControl.SetPDClient(tc, pdClient)

		if test.deleteMemberErr {
			pdClient.AddReaction(controller.DeleteMemberActionType, func(action *controller.Action) (interface{}, error) {
				return nil, fmt.Errorf("error")
			})
		}
		if test.pvcUpdateErr {
			pvcControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := scaler.ScaleIn(tc, oldSet, newSet)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(4))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:            "normal",
			pdUpgrading:     false,
			hasPVC:          true,
			pvcUpdateErr:    false,
			deleteMemberErr: false,
			err:             false,
			changed:         true,
		},
		{
			name:            "pd is upgrading",
			pdUpgrading:     true,
			hasPVC:          true,
			pvcUpdateErr:    false,
			deleteMemberErr: false,
			err:             false,
			changed:         false,
		},
		{
			name:            "error when delete member",
			hasPVC:          true,
			pvcUpdateErr:    false,
			pdUpgrading:     false,
			deleteMemberErr: true,
			err:             true,
			changed:         false,
		},
		{
			name:            "cache don't have pvc",
			pdUpgrading:     false,
			hasPVC:          false,
			pvcUpdateErr:    false,
			deleteMemberErr: false,
			err:             true,
			changed:         false,
		},
		{
			name:            "error when update pvc",
			pdUpgrading:     false,
			hasPVC:          true,
			pvcUpdateErr:    true,
			deleteMemberErr: false,
			err:             true,
			changed:         false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakePDScaler() (*pdScaler, *controller.FakePDControl, cache.Indexer, *controller.FakePVCControl) {
	kubeCli := kubefake.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pdControl := controller.NewFakePDControl()
	pvcControl := controller.NewFakePVCControl(pvcInformer)

	return &pdScaler{pdControl, pvcInformer.Lister(), pvcControl},
		pdControl, pvcInformer.Informer().GetIndexer(), pvcControl
}

func newStatefulSetForPDScale() *apps.StatefulSet {
	set := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scaler",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: int32Pointer(5),
		},
	}
	return set
}

func newPVCForStatefulSet(set *apps.StatefulSet) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pd-%s-%d", set.GetName(), int(*set.Spec.Replicas)-1),
			Namespace: metav1.NamespaceDefault,
		},
	}
}

func int32Pointer(num int) *int32 {
	i := int32(num)
	return &i
}
