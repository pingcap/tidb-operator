// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1alpha1validation "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/validation"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/manager/tidbdashboard"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clitesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestReconcile(t *testing.T) {

	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		getTC                func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster
		validateErr          error
		syncReclaimPolicyErr error
		syncDashboard        func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error
		syncTCTLSCerts       func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error
		updateStatusErr      error
		errExpectFn          func(error)
	}

	cases := []testcase{
		{
			name: "reconcile succeeded",
			getTC: func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster {
				tc := &v1alpha1.TidbCluster{}
				tc.Name = "tc"
				tc.Namespace = "default"
				td.Spec.Clusters = []v1alpha1.TidbClusterRef{{Name: "tc", Namespace: "default"}}
				return tc
			},
			errExpectFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name:        "validate failed",
			validateErr: fmt.Errorf("validate error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name: "sync reclaim policy failed",
			getTC: func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster {
				tc := &v1alpha1.TidbCluster{}
				tc.Name = "tc"
				tc.Namespace = "default"
				td.Spec.Clusters = []v1alpha1.TidbClusterRef{{Name: "tc", Namespace: "default"}}
				return tc
			},
			syncReclaimPolicyErr: fmt.Errorf("sync reclaim policy error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("sync reclaim policy error"))
			},
		},
		{
			name: "tc is not found",
			getTC: func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster {
				td.Spec.Clusters = []v1alpha1.TidbClusterRef{{Name: "tc", Namespace: "default"}}
				return nil
			},
			syncReclaimPolicyErr: fmt.Errorf("sync reclaim policy error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("not found"))
			},
		},
		{
			name: "sync dashboard failed",
			getTC: func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster {
				tc := &v1alpha1.TidbCluster{}
				tc.Name = "tc"
				tc.Namespace = "default"
				td.Spec.Clusters = []v1alpha1.TidbClusterRef{{Name: "tc", Namespace: "default"}}
				return tc
			},
			syncDashboard: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
				return fmt.Errorf("sync dashboard error")
			},
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("sync dashboard error"))
			},
		},
		{
			name: "sync tc tls certs failed",
			getTC: func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster {
				tc := &v1alpha1.TidbCluster{}
				tc.Name = "tc"
				tc.Namespace = "default"
				td.Spec.Clusters = []v1alpha1.TidbClusterRef{{Name: "tc", Namespace: "default"}}
				return tc
			},
			syncTCTLSCerts: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
				return fmt.Errorf("sync tc tls certs error")
			},
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("sync tc tls certs error"))
			},
		},
		{
			name: "status of TidbDashboard have not changed",
			getTC: func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster {
				tc := &v1alpha1.TidbCluster{}
				tc.Name = "tc"
				tc.Namespace = "default"
				td.Spec.Clusters = []v1alpha1.TidbClusterRef{{Name: "tc", Namespace: "default"}}
				return tc
			},
			updateStatusErr: fmt.Errorf("updateStatus TidbDashboard error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name: "updateStatus TidbDashboard failed",
			getTC: func(td *v1alpha1.TidbDashboard) *v1alpha1.TidbCluster {
				tc := &v1alpha1.TidbCluster{}
				tc.Name = "tc"
				tc.Namespace = "default"
				td.Spec.Clusters = []v1alpha1.TidbClusterRef{{Name: "tc", Namespace: "default"}}
				return tc
			},
			syncDashboard: func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
				td.Status.Synced = true
				return nil
			},
			updateStatusErr: fmt.Errorf("updateStatus TidbDashboard error"),
			errExpectFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("updateStatus TidbDashboard error"))
			},
		},
	}

	runTestcase := func(testcase testcase) {
		t.Logf("testcase: %s", testcase.name)

		control, deps := newTidbDashboardControlForTest()
		tcIndexer := deps.InformerFactory.Pingcap().V1alpha1().TidbClusters().Informer().GetIndexer()

		td := newTidbDashboardForTest()

		validatePatch := gomonkey.ApplyFunc(v1alpha1validation.ValidateTiDBDashboard, func(td *v1alpha1.TidbDashboard) field.ErrorList {
			allErrs := field.ErrorList{}
			if testcase.validateErr != nil {
				allErrs = append(allErrs, field.InternalError(field.NewPath(""), testcase.validateErr))
			}
			return allErrs
		})
		defer validatePatch.Reset()

		if testcase.getTC != nil {
			if tc := testcase.getTC(td); tc != nil {
				tcIndexer.Add(tc)
			}
		}

		if testcase.syncReclaimPolicyErr != nil {
			control.reclaimPolicyManager.(*meta.FakeReclaimPolicyManager).SetSyncError(testcase.syncReclaimPolicyErr)
		}

		if testcase.syncDashboard != nil {
			control.dashboardManager.(*tidbdashboard.FakeManager).MockSync(testcase.syncDashboard)
		}

		if testcase.syncTCTLSCerts != nil {
			control.dashboardManager.(*tidbdashboard.FakeManager).MockSync(testcase.syncTCTLSCerts)
		}

		updateStatusPatch := gomonkey.ApplyPrivateMethod(reflect.TypeOf(control), "updateStatus", func(_ *defaultTiDBDashboardControl, td *v1alpha1.TidbDashboard) (*v1alpha1.TidbDashboard, error) {
			return td, testcase.updateStatusErr
		})
		defer updateStatusPatch.Reset()

		err := control.Reconcile(td)
		testcase.errExpectFn(err)
	}

	for _, testcase := range cases {
		runTestcase(testcase)
	}
}

func TestUpdateStatus(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		updateFn func(clitesting.Action) (runtime.Object, error)

		expectErrFn func(error)
	}

	conflict := true
	cases := []testcase{
		{
			name: "updateStatus succeeded",
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name: "retry when conflict",
			updateFn: func(action clitesting.Action) (runtime.Object, error) {
				update := action.(clitesting.UpdateAction)
				td := update.GetObject().(*v1alpha1.TidbDashboard)

				if conflict {
					conflict = false
					return td, apierrors.NewConflict(action.GetResource().GroupResource(), td.Name, errors.New("conflict"))
				}
				return td, nil
			},
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name: "updateStatus failed",
			updateFn: func(action clitesting.Action) (runtime.Object, error) {
				return action.(clitesting.UpdateAction).GetObject(), fmt.Errorf("updateStatus error")
			},
			expectErrFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("updateStatus error"))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		control, _ := newTidbDashboardControlForTest()
		td := newTidbDashboardForTest()

		control.deps.Clientset.(*fake.Clientset).PrependReactor("update", v1alpha1.TiDBDashboardName, func(action clitesting.Action) (bool, runtime.Object, error) {
			if testcase.updateFn != nil {
				td, err := testcase.updateFn(action)
				return true, td, err
			}
			return true, action.(clitesting.UpdateAction).GetObject(), nil
		})

		_, err := control.updateStatus(td)
		testcase.expectErrFn(err)
	}
}

func newTidbDashboardControlForTest() (*defaultTiDBDashboardControl, *controller.Dependencies) {
	deps := controller.NewFakeDependencies()
	recorder := record.NewFakeRecorder(10)

	tdManager := tidbdashboard.NewFakeManager()
	tlsManager := tidbdashboard.NewFakeManager()
	reclaimPolicyManager := meta.NewFakeReclaimPolicyManager()

	control := &defaultTiDBDashboardControl{
		deps:                 deps,
		recorder:             recorder,
		dashboardManager:     tdManager,
		tlsCertManager:       tlsManager,
		reclaimPolicyManager: reclaimPolicyManager,
	}

	return control, deps
}

func newTidbDashboardForTest() *v1alpha1.TidbDashboard {
	return &v1alpha1.TidbDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       "test",
		},
		Spec: v1alpha1.TidbDashboardSpec{},
	}
}
