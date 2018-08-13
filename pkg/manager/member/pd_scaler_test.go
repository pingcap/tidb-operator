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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPDScalerScaleDown(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		modify          func(*apps.StatefulSet) *apps.StatefulSet
		pdUpgrading     bool
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

		oldSet := newStatefulSetForPDScaleDown()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = int32Pointer(3)

		scaler, pdControl := newFakePDScaler()
		pdClient := controller.NewFakePDClient()
		pdControl.SetPDClient(tc, pdClient)

		if test.deleteMemberErr {
			pdClient.AddReaction(controller.DeleteMemberActionType, func(action *controller.Action) (interface{}, error) {
				return nil, fmt.Errorf("error")
			})
		}

		err := scaler.ScaleDown(tc, oldSet, newSet)
		if test.err {
			g.Expect(err).To(HaveOccurred())
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
			deleteMemberErr: false,
			err:             false,
			changed:         true,
		},
		{
			name:            "pd is upgrading",
			pdUpgrading:     true,
			deleteMemberErr: false,
			err:             false,
			changed:         false,
		},
		{
			name:            "error when delete member",
			pdUpgrading:     false,
			deleteMemberErr: true,
			err:             true,
			changed:         false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakePDScaler() (*pdScaler, *controller.FakePDControl) {
	pdControl := controller.NewFakePDControl()
	return &pdScaler{pdControl}, pdControl
}

func newStatefulSetForPDScaleDown() *apps.StatefulSet {
	set := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pd-scale-down",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: int32Pointer(5),
		},
	}
	return set
}

func int32Pointer(num int) *int32 {
	i := int32(num)
	return &i
}
