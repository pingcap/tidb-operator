// Copyright 2019 PingCAP, Inc.
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
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestTidbDiscoveryManager_Reconcile(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		prepare             func(tc *v1alpha1.TidbCluster, ctrl *controller.FakeGenericControl)
		errOnCreateOrUpdate bool
		expect              func([]appsv1.Deployment, *v1alpha1.TidbCluster, error)
	}
	testFn := func(tt *testcase) {
		t.Log(tt.name)

		tc := newTidbClusterForTiDB()
		dm, ctrl := newFakeTidbDiscoveryManager()
		if tt.prepare != nil {
			tt.prepare(tc, ctrl)
		}
		if tt.errOnCreateOrUpdate {
			ctrl.SetCreateOrUpdateError(fmt.Errorf("API server down"), 0)
		}
		err := dm.Reconcile(tc)
		deployList := &appsv1.DeploymentList{}
		_ = ctrl.FakeCli.List(context.TODO(), deployList)
		tt.expect(deployList.Items, tc, err)
	}

	cases := []*testcase{
		{
			name: "Basic",
			expect: func(deploys []appsv1.Deployment, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(deploys).To(HaveLen(1))
				g.Expect(deploys[0].Name).To((Equal("test-discovery")))
			},
			errOnCreateOrUpdate: false,
		},
		{
			name: "Setting discovery resource",
			prepare: func(tc *v1alpha1.TidbCluster, ctrl *controller.FakeGenericControl) {
				tc.Spec.Discovery.ResourceRequirements = corev1.ResourceRequirements{

					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("1"),
						corev1.ResourceMemory:           resource.MustParse("2Gi"),
						corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
					},
				}
			},
			expect: func(deploys []appsv1.Deployment, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(deploys).To(HaveLen(1))
				g.Expect(deploys[0].Name).To((Equal("test-discovery")))
			},
			errOnCreateOrUpdate: false,
		},
		{
			name: "Create or update resource error",
			expect: func(deploys []appsv1.Deployment, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(Succeed())
				g.Expect(deploys).To(BeEmpty())
			},
			errOnCreateOrUpdate: true,
		},
	}
	for _, tt := range cases {
		testFn(tt)
	}
}

func newFakeTidbDiscoveryManager() (*realTidbDiscoveryManager, *controller.FakeGenericControl) {
	ctrl := controller.NewFakeGenericControl()
	return &realTidbDiscoveryManager{
		ctrl: controller.NewTypedControl(ctrl),
	}, ctrl
}
