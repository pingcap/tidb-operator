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

package tidbngmonitoring

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1alpha1validation "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/validation"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/manager/tidbngmonitoring"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clitesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestReconcile(t *testing.T) {

	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		updateTNGM func(tngm *v1alpha1.TidbNGMonitoring)

		validateErr          error
		syncReclaimPolicyErr error
		syncNGMonitoring     func(tngm *v1alpha1.TidbNGMonitoring) error
		updeteTNGMErr        error
		errExpectFn          func(error)
	}

	cases := []testcase{
		{
			name: "reconcile succeeded",
			errExpectFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name:        "validate failed",
			validateErr: fmt.Errorf("validate error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(Succeed()) // return nil if validation is failed
			},
		},
		{
			name:                 "sync reclaim policy failed",
			syncReclaimPolicyErr: fmt.Errorf("sync reclaim policy error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("sync reclaim policy error"))
			},
		},
		{
			name: "sync ng mmonitoring failed",
			syncNGMonitoring: func(tngm *v1alpha1.TidbNGMonitoring) error {
				return fmt.Errorf("sync ng mmonitoring error")
			},
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("sync ng mmonitoring error"))
			},
		},
		{
			name:          "status of TidbNGMonitoring have not changed",
			updateTNGM:    nil,
			updeteTNGMErr: fmt.Errorf("update TidbNGMonitoring error"), // shouldn't return this err
			errExpectFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name:       "update TidbNGMonitoring failed",
			updateTNGM: nil,
			syncNGMonitoring: func(tngm *v1alpha1.TidbNGMonitoring) error {
				tngm.Status.NGMonitoring.Synced = true
				return nil
			},
			updeteTNGMErr: fmt.Errorf("update TidbNGMonitoring error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("update TidbNGMonitoring error"))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		control := newTiDBNGMonitoringControlForTest()

		tngm := newTidbNGMonitoringForTest()
		if testcase.updateTNGM != nil {
			testcase.updateTNGM(tngm)
		}

		// mock result of validation
		validatePatch := gomonkey.ApplyFunc(v1alpha1validation.ValidateTiDBNGMonitoring, func(tngm *v1alpha1.TidbNGMonitoring) field.ErrorList {
			allErrs := field.ErrorList{}
			if testcase.validateErr != nil {
				allErrs = append(allErrs, field.InternalError(field.NewPath(""), testcase.validateErr))
			}
			return allErrs
		})
		defer validatePatch.Reset()

		// mock result of reclaim policy synchronization
		if testcase.syncReclaimPolicyErr != nil {
			control.reclaimPolicyManager.(*meta.FakeReclaimPolicyManager).SetSyncError(testcase.syncReclaimPolicyErr)
		}

		// mock result of ng monitoring synchronization
		if testcase.syncNGMonitoring != nil {
			control.ngmMnger.(*tidbngmonitoring.FakeNGMonitoringManager).MockSync(testcase.syncNGMonitoring)
		}

		// mock result of updating TidbNGMonitoring
		updateTNGMPatch := gomonkey.ApplyMethod(reflect.TypeOf(control), "Update", func(_ *defaultTiDBNGMonitoringControl, tngm *v1alpha1.TidbNGMonitoring) (*v1alpha1.TidbNGMonitoring, error) {
			return tngm, testcase.updeteTNGMErr
		})
		defer updateTNGMPatch.Reset()

		err := control.Reconcile(tngm)
		testcase.errExpectFn(err)
	}
}

func TestUpdate(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		updateFn func(clitesting.Action) (runtime.Object, error)

		expectErrFn func(error)
	}

	conflict := true
	cases := []testcase{
		{
			name: "update succeeded",
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name: "retry when conflict",
			updateFn: func(action clitesting.Action) (runtime.Object, error) {
				update := action.(clitesting.UpdateAction)
				tnmg := update.GetObject().(*v1alpha1.TidbNGMonitoring)

				if conflict {
					conflict = false
					return tnmg, apierrors.NewConflict(action.GetResource().GroupResource(), tnmg.Name, errors.New("conflict"))
				}
				return tnmg, nil
			},
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name: "update failed",
			updateFn: func(action clitesting.Action) (runtime.Object, error) {
				return action.(clitesting.UpdateAction).GetObject(), fmt.Errorf("update error")
			},
			expectErrFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("update error"))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		control := newTiDBNGMonitoringControlForTest()
		tngm := newTidbNGMonitoringForTest()

		control.cli.(*fake.Clientset).PrependReactor("update", v1alpha1.TiDBNGMonitoringName, func(action clitesting.Action) (bool, runtime.Object, error) {
			if testcase.updateFn != nil {
				tngm, err := testcase.updateFn(action)
				return true, tngm, err
			}
			return true, action.(clitesting.UpdateAction).GetObject(), nil
		})

		_, err := control.Update(tngm)
		testcase.expectErrFn(err)
	}
}

func newTiDBNGMonitoringControlForTest() *defaultTiDBNGMonitoringControl {
	cli := fake.NewSimpleClientset()
	tngmInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbNGMonitorings()
	recorder := record.NewFakeRecorder(10)

	ngmManager := tidbngmonitoring.NewFakeNGMonitoringManager()
	reclaimPolicyManager := meta.NewFakeReclaimPolicyManager()

	control := NewDefaultTiDBNGMonitoringControl(
		cli,
		tngmInformer.Lister(),
		ngmManager,
		reclaimPolicyManager,
		recorder,
	)

	return control
}

func newTidbNGMonitoringForTest() *v1alpha1.TidbNGMonitoring {
	return &v1alpha1.TidbNGMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbNGMonitoringSpec{
			NGMonitoring: v1alpha1.NGMonitoringSpec{},
		},
	}
}
