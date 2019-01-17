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
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestPDStatusSyncerSyncStatefulSetStatus(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		set      *apps.StatefulSet
		expectFn func(*GomegaWithT, *v1alpha1.TidbCluster, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForPD()
		syncer, setIndexer, _ := newFakePDStatusSyncer()
		if test.set != nil {
			setIndexer.Add(test.set)
		}
		test.expectFn(g, tc, syncer.SyncStatefulSetStatus(tc))
	}
	tests := []testcase{
		{
			name: "normal",
			set: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pd",
					Namespace: corev1.NamespaceDefault,
				},
				Status: apps.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					CurrentRevision: "pd-1",
				},
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				status := &apps.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					CurrentRevision: "pd-1",
				}
				g.Expect(tc.Status.PD.StatefulSet).To(Equal(status))
			},
		},
		{
			name: "doesn't have statefulset",
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "statefulset.apps \"test-pd\" not found")).To(BeTrue())
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDStatusSyncerSyncExtraStatefulSetsStatus(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		update   func(tc *v1alpha1.TidbCluster)
		sets     []*apps.StatefulSet
		expectFn func(*GomegaWithT, *v1alpha1.TidbCluster, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForPD()
		syncer, setIndexer, _ := newFakePDStatusSyncer()
		for _, set := range test.sets {
			setIndexer.Add(set)
		}
		if test.update != nil {
			test.update(tc)
		}
		test.expectFn(g, tc, syncer.SyncExtraStatefulSetsStatus(tc))
	}
	tests := []testcase{
		{
			name: "pds is empty",
			sets: []*apps.StatefulSet{
				extraSet("first"),
				extraSet("second"),
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(tc.Status.PD.StatefulSets)).To(Equal(0))
			},
		},
		{
			name: "have the first pds",
			sets: []*apps.StatefulSet{
				extraSet("first"),
				extraSet("second"),
			},
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{Name: "first"},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(tc.Status.PD.StatefulSets)).To(Equal(1))
				status := apps.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					CurrentRevision: "first-pd-1",
				}
				g.Expect(tc.Status.PD.StatefulSets["test-first-pd"]).To(Equal(status))
			},
		},
		{
			name: "have the second pds",
			sets: []*apps.StatefulSet{
				extraSet("first"),
				extraSet("second"),
			},
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{Name: "second"},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(tc.Status.PD.StatefulSets)).To(Equal(1))
				status := apps.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					CurrentRevision: "second-pd-1",
				}
				g.Expect(tc.Status.PD.StatefulSets["test-second-pd"]).To(Equal(status))
			},
		},
		{
			name: "have both pds",
			sets: []*apps.StatefulSet{
				extraSet("first"),
				extraSet("second"),
			},
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{Name: "first"},
					{Name: "second"},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(tc.Status.PD.StatefulSets)).To(Equal(2))
				statusFirst := apps.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					CurrentRevision: "first-pd-1",
				}
				statusSecond := apps.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					CurrentRevision: "second-pd-1",
				}
				g.Expect(tc.Status.PD.StatefulSets["test-first-pd"]).To(Equal(statusFirst))
				g.Expect(tc.Status.PD.StatefulSets["test-second-pd"]).To(Equal(statusSecond))
			},
		},
		{
			name: "doesn't have second statefulset",
			sets: []*apps.StatefulSet{
				extraSet("first"),
			},
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{Name: "first"},
					{Name: "second"},
				}
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "statefulset.apps \"test-second-pd\" not found")).To(BeTrue())
				g.Expect(len(tc.Status.PD.StatefulSets)).To(Equal(1))
				statusFirst := apps.StatefulSetStatus{
					Replicas:        1,
					ReadyReplicas:   1,
					CurrentRevision: "first-pd-1",
				}
				g.Expect(tc.Status.PD.StatefulSets["test-first-pd"]).To(Equal(statusFirst))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDStatusSyncerSyncPDMembers(t *testing.T) {
	g := NewGomegaWithT(t)
	time1 := metav1.Now()
	time2 := metav1.Now()

	type testcase struct {
		name          string
		update        func(tc *v1alpha1.TidbCluster)
		getHealthFn   func() (*controller.HealthInfo, error)
		getPDLeaderFn func() (*pdpb.Member, error)
		expectFn      func(*GomegaWithT, *v1alpha1.TidbCluster, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForPD()
		if test.update != nil {
			test.update(tc)
		}
		syncer, _, fakePDControl := newFakePDStatusSyncer()
		pdClient := controller.NewFakePDClient()
		fakePDControl.SetPDClient(tc, pdClient)
		pdClient.AddReaction(controller.GetHealthActionType, func(action *controller.Action) (interface{}, error) {
			return test.getHealthFn()
		})
		pdClient.AddReaction(controller.GetPDLeaderActionType, func(action *controller.Action) (interface{}, error) {
			return test.getPDLeaderFn()
		})

		test.expectFn(g, tc, syncer.SyncPDMembers(tc))
	}
	tests := []testcase{
		{
			name: "failed to getHealth",
			getHealthFn: func() (*controller.HealthInfo, error) {
				return nil, fmt.Errorf("failed to getHealth")
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to getHealth")).To(BeTrue())
				g.Expect(tc.Status.PD.Synced).To(BeFalse())
			},
		},
		{
			name: "failed to getPDLeader",
			getHealthFn: func() (*controller.HealthInfo, error) {
				return nil, nil
			},
			getPDLeaderFn: func() (*pdpb.Member, error) {
				return nil, fmt.Errorf("failed to getPDLeader")
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to getPDLeader")).To(BeTrue())
				g.Expect(tc.Status.PD.Synced).To(BeFalse())
			},
		},
		{
			name: "healthInfo is empty",
			getHealthFn: func() (*controller.HealthInfo, error) {
				return &controller.HealthInfo{}, nil
			},
			getPDLeaderFn: func() (*pdpb.Member, error) {
				return nil, nil
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(tc.Status.PD.Members)).To(Equal(0))
				g.Expect(tc.Status.PD.Synced).To(BeTrue())
			},
		},
		{
			name: "normal",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.Members = map[string]v1alpha1.PDMember{
					"test-pd-0": {
						Name:               "test-pd-0",
						Health:             true,
						LastTransitionTime: time1,
					},
					"test-pd-1": {
						Name:               "test-pd-1",
						Health:             true,
						LastTransitionTime: time2,
					},
				}
			},
			getHealthFn: func() (*controller.HealthInfo, error) {
				return &controller.HealthInfo{
					Healths: []controller.MemberHealth{
						{
							Name:       "test-pd-0",
							MemberID:   uint64(1),
							ClientUrls: []string{"test-pd-0"},
							Health:     true,
						},
						{
							Name:       "test-pd-1",
							MemberID:   uint64(2),
							ClientUrls: []string{"test-pd-1"},
							Health:     false,
						},
						{
							Name:       "test-pd-2",
							MemberID:   uint64(3),
							ClientUrls: []string{"test-pd-2"},
							Health:     true,
						},
						{
							MemberID:   uint64(4),
							ClientUrls: []string{"test-pd-3"},
							Health:     true,
						},
					},
				}, nil
			},
			getPDLeaderFn: func() (*pdpb.Member, error) {
				return &pdpb.Member{Name: "test-pd-0"}, nil
			},
			expectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(tc.Status.PD.Members)).To(Equal(3))
				g.Expect(tc.Status.PD.Members["test-pd-0"].Name).To(Equal("test-pd-0"))
				g.Expect(tc.Status.PD.Members["test-pd-0"].ID).To(Equal("1"))
				g.Expect(tc.Status.PD.Members["test-pd-0"].ClientURL).To(Equal("test-pd-0"))
				g.Expect(tc.Status.PD.Members["test-pd-0"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Members["test-pd-0"].LastTransitionTime).To(Equal(time1))
				g.Expect(tc.Status.PD.Members["test-pd-1"].Name).To(Equal("test-pd-1"))
				g.Expect(tc.Status.PD.Members["test-pd-1"].ID).To(Equal("2"))
				g.Expect(tc.Status.PD.Members["test-pd-1"].ClientURL).To(Equal("test-pd-1"))
				g.Expect(tc.Status.PD.Members["test-pd-1"].Health).To(Equal(false))
				g.Expect(tc.Status.PD.Members["test-pd-1"].LastTransitionTime).NotTo(Equal(time2))
				g.Expect(tc.Status.PD.Members["test-pd-2"].Name).To(Equal("test-pd-2"))
				g.Expect(tc.Status.PD.Members["test-pd-2"].ID).To(Equal("3"))
				g.Expect(tc.Status.PD.Members["test-pd-2"].ClientURL).To(Equal("test-pd-2"))
				g.Expect(tc.Status.PD.Members["test-pd-2"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Synced).To(BeTrue())
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakePDStatusSyncer() (*pdStatusSyncer, cache.Indexer, *controller.FakePDControl) {
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	pdControl := controller.NewFakePDControl()

	return &pdStatusSyncer{
		setInformer.Lister(),
		pdControl,
	}, setInformer.Informer().GetIndexer(), pdControl
}

func extraSet(name string) *apps.StatefulSet {
	return &apps.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("test-%s-pd", name),
			Namespace: corev1.NamespaceDefault,
		},
		Status: apps.StatefulSetStatus{
			Replicas:        1,
			ReadyReplicas:   1,
			CurrentRevision: fmt.Sprintf("%s-pd-1", name),
		},
	}
}
