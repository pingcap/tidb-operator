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
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFakeTiDBFailoverFailover(t *testing.T) {
	type testcase struct {
		name        string
		update      func(*v1alpha1.TidbCluster)
		tcUpdateErr bool
		errExpectFn func(*GomegaWithT, error)
		expectFn    func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Logf(test.name)
		g := NewGomegaWithT(t)
		tidbFailover, tcControl := newTiDBFailover()
		tc := newTidbClusterForTiDBFailover()
		test.update(tc)
		if test.tcUpdateErr {
			tcControl.SetUpdateTidbClusterError(fmt.Errorf("update tidbcluster error"), 0)
		}

		err := tidbFailover.Failover(tc)
		test.errExpectFn(g, err)
		test.expectFn(g, tc)
	}

	tests := []testcase{
		{
			name: "all tidb members are ready",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			tcUpdateErr: false,
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "one tidb member failed",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			tcUpdateErr: false,
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(1))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(3))
			},
		},
		{
			name: "two tidb members failed",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: false,
					},
				}
			},
			tcUpdateErr: false,
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(1))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(3))
			},
		},
		{
			name: "all tidb members are ready and error should happened when update tidbcluster",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			tcUpdateErr: true,
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).NotTo(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "one tidb member failed and error should happened when update tidbcluster",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: false,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			tcUpdateErr: true,
			errExpectFn: func(t *GomegaWithT, err error) {
				t.Expect(err).To(HaveOccurred())
			},
			expectFn: func(t *GomegaWithT, tc *v1alpha1.TidbCluster) {
				t.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestFakeTiDBFailoverRecover(t *testing.T) {
	type testcase struct {
		name     string
		update   func(*v1alpha1.TidbCluster)
		expectFn func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		g := NewGomegaWithT(t)
		tidbFailover, _ := newTiDBFailover()
		tc := newTidbClusterForTiDBFailover()
		test.update(tc)

		tidbFailover.Recover(tc)
		test.expectFn(g, tc)
	}

	tests := []testcase{
		{
			name: "have not failure tidb member to recover",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
			},
		},
		{
			name: "one failure tidb member to recover",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 3
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName:  "failover-tidb-0",
						Replicas: 2,
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure tidb members to recover",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 4
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: true,
					},
					"failover-tidb-3": {
						Name:   "failover-tidb-3",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName:  "failover-tidb-0",
						Replicas: 2,
					},
					"failover-tidb-1": {
						PodName:  "failover-tidb-0",
						Replicas: 3,
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(2))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure tidb members to recover and user have set a larger replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 5
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
					"failover-tidb-1": {
						Name:   "failover-tidb-1",
						Health: true,
					},
					"failover-tidb-2": {
						Name:   "failover-tidb-2",
						Health: true,
					},
					"failover-tidb-3": {
						Name:   "failover-tidb-3",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName:  "failover-tidb-0",
						Replicas: 2,
					},
					"failover-tidb-1": {
						PodName:  "failover-tidb-0",
						Replicas: 3,
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(5))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
		{
			name: "two failure tidb members to recover and user have set a smaller replicas",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiDB.Replicas = 1
				tc.Status.TiDB.Members = map[string]v1alpha1.TiDBMember{
					"failover-tidb-0": {
						Name:   "failover-tidb-0",
						Health: true,
					},
				}
				tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{
					"failover-tidb-0": {
						PodName:  "failover-tidb-0",
						Replicas: 2,
					},
					"failover-tidb-1": {
						PodName:  "failover-tidb-0",
						Replicas: 3,
					},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(int(tc.Spec.TiDB.Replicas)).To(Equal(1))
				g.Expect(len(tc.Status.TiDB.FailureMembers)).To(Equal(0))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}

}

func newTiDBFailover() (Failover, *controller.FakeTidbClusterControl) {
	cli := fake.NewSimpleClientset()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	tcControl := controller.NewFakeTidbClusterControl(tcInformer)
	return &tidbFailover{tidbFailoverPeriod: time.Duration(5 * time.Minute), tcControl: tcControl}, tcControl
}

func newTidbClusterForTiDBFailover() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failover",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("failover"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiDB: v1alpha1.TiDBSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "tidb-test-image",
				},
				Replicas:         2,
				StorageClassName: "my-storage-class",
			},
		},
	}
}
