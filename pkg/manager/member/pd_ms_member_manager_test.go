// Copyright 2023 PingCAP, Inc.
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
	"github.com/google/go-cmp/cmp"
	"github.com/pingcap/kvproto/pkg/metapb"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestPDMSMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                       string
		errWhenCreateStatefulSet   bool
		errWhenCreatePDService     bool
		errWhenCreatePDPeerService bool
		statusChange               func(*apps.StatefulSet)
		suspendComponent           func() (bool, error)
		errExpectFn                func(*GomegaWithT, error)
		setCreated                 bool
		tls                        bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPDMS()
		if test.tls {
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
		}
		ns := tc.Namespace
		tcName := tc.Name
		oldSpec := tc.Spec

		// finish pd cluster
		pdmm, _, _ := newFakePDMemberManager()
		fakeSetControl := pdmm.deps.StatefulSetControl.(*controller.FakeStatefulSetControl)
		fakeSvcControl := pdmm.deps.ServiceControl.(*controller.FakeServiceControl)
		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreatePDService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreatePDPeerService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 1)
		}
		if test.suspendComponent != nil {
			pdmm.suspender.(*suspender.FakeSuspender).SuspendComponentFunc = func(c v1alpha1.Cluster, mt v1alpha1.MemberType) (bool, error) {
				return test.suspendComponent()
			}
		}

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "pd-1"
				set.Status.UpdateRevision = "pd-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
				set.Status.ReadyReplicas = 1
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		// retry Sync, wait for pd cluster ready.
		fakePDControl := pdmm.deps.PDControl.(*pdapi.FakePDControl)
		pdClient := controller.NewFakePDClient(fakePDControl, tc)
		pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
			return &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://test-pd-1.test-pd-peer.default.svc:2379"}, Health: true},
			}}, nil
		})
		pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
			return &metapb.Cluster{Id: uint64(1)}, nil
		})
		err := pdmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		err = pdmm.Sync(tc)

		tc1, err := pdmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
		if test.setCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(tc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		pmm, _, _ := newFakePDMSMemberManager(tc.Spec.PDMS)
		fakeSetControl = pmm.deps.StatefulSetControl.(*controller.FakeStatefulSetControl)
		fakeSvcControl = pmm.deps.ServiceControl.(*controller.FakeServiceControl)
		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreatePDService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreatePDPeerService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 1)
		}
		if test.suspendComponent != nil {
			pmm.suspender.(*suspender.FakeSuspender).SuspendComponentFunc = func(c v1alpha1.Cluster, mt v1alpha1.MemberType) (bool, error) {
				return test.suspendComponent()
			}
		}

		err = pmm.Sync(tc)
		test.errExpectFn(g, err)
		g.Expect(tc.Spec).To(Equal(oldSpec))

		tc1, err = pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, "tso"))
		if test.setCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(tc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}
	}

	tests := []testcase{
		{
			name:                       "normal",
			errWhenCreateStatefulSet:   false,
			errWhenCreatePDService:     false,
			errWhenCreatePDPeerService: false,
			errExpectFn:                errExpectRequeue,
			setCreated:                 true,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDMSMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                       string
		modify                     func(cluster *v1alpha1.TidbCluster)
		pdHealth                   *pdapi.HealthInfo
		errWhenUpdateStatefulSet   bool
		errWhenUpdatePDService     bool
		errWhenUpdatePDPeerService bool
		errWhenGetCluster          bool
		errWhenGetPDHealth         bool
		statusChange               func(*apps.StatefulSet)
		err                        bool
		expectPDServiceFn          func(*GomegaWithT, *corev1.Service, error)
		expectPDPeerServiceFn      func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn        func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn        func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPDMS()
		ns := tc.Namespace
		tcName := tc.Name

		pdmm, _, _ := newFakePDMemberManager()
		fakePDControl := pdmm.deps.PDControl.(*pdapi.FakePDControl)
		fakeSetControl := pdmm.deps.StatefulSetControl.(*controller.FakeStatefulSetControl)
		//fakeSvcControl := pdmm.deps.ServiceControl.(*controller.FakeServiceControl)
		pdClient := controller.NewFakePDClient(fakePDControl, tc)
		if test.errWhenGetPDHealth {
			pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get health of pd cluster")
			})
		} else {
			pdClient.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
				return test.pdHealth, nil
			})
		}

		if test.errWhenGetCluster {
			pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get cluster info")
			})
		} else {
			pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
				return &metapb.Cluster{Id: uint64(1)}, nil
			})
		}

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "pd-1"
				set.Status.UpdateRevision = "pd-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		err := pdmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		err = pdmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectPDServiceFn != nil {
			svc, err := pdmm.deps.ServiceLister.Services(ns).Get(controller.PDMemberName(tcName))
			test.expectPDServiceFn(g, svc, err)
		}
		if test.expectPDPeerServiceFn != nil {
			svc, err := pdmm.deps.ServiceLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
			test.expectPDPeerServiceFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := pdmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc)
		}

		// pd ms
		pmm, _, _ := newFakePDMSMemberManager(tc.Spec.PDMS)
		err = pmm.Sync(tc)

		tc5, err := pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, "tso"))
		test.expectStatefulSetFn(g, tc5, err)
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
				tc.Spec.Services = []v1alpha1.Service{
					{Name: "pd", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://test-pd-1.test-pd-peer.default.svc:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://test-pd-2.test-pd-peer.default.svc:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://test-pd-3.test-pd-peer.default.svc:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetPDHealth:         false,
			err:                        false,
			expectPDServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				// g.Expect(int(*set.Spec.Replicas)).To(Equal(4))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.ClusterID).To(Equal("1"))
				g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.ScalePhase))
				g.Expect(tc.Status.PD.StatefulSet.ObservedGeneration).To(Equal(int64(1)))
				g.Expect(len(tc.Status.PD.Members)).To(Equal(3))
				g.Expect(tc.Status.PD.Members["pd1"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Members["pd2"].Health).To(Equal(true))
				g.Expect(tc.Status.PD.Members["pd3"].Health).To(Equal(false))
			},
		},
		{
			name: "error when update pd service",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.Services = []v1alpha1.Service{
					{Name: "pd", Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     true,
			errWhenUpdatePDPeerService: false,
			err:                        true,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn:        nil,
		},
		{
			name: "error when update statefulset",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://pd3:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   true,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err:                        true,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "error when sync pd status",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetPDHealth:         true,
			err:                        false,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.Synced).To(BeFalse())
				g.Expect(tc.Status.PD.Members).To(BeNil())
			},
		},
		{
			name: "error when sync cluster ID",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetCluster:          true,
			errWhenGetPDHealth:         false,
			err:                        false,
			expectPDServiceFn:          nil,
			expectPDPeerServiceFn:      nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PD.Synced).To(BeFalse())
				g.Expect(tc.Status.PD.Members).To(BeNil())
			},
		},
		{
			name: "patch pd container lifecycle configuration when sync cluster  ",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
				tc.Spec.PD.AdditionalContainers = []corev1.Container{
					{Name: "pd", Lifecycle: &corev1.Lifecycle{PreStop: &corev1.LifecycleHandler{
						Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "echo 'test'"}},
					}}},
				}
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://test-pd-1.test-pd-peer.default.svc:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://test-pd-2.test-pd-peer.default.svc:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://test-pd-3.test-pd-peer.default.svc:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetPDHealth:         false,
			err:                        false,
			expectPDServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Lifecycle).To(Equal(
					&corev1.Lifecycle{PreStop: &corev1.LifecycleHandler{
						Exec: &corev1.ExecAction{Command: []string{"sh", "-c", "echo 'test'"}},
					}}))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
			},
		},
		{
			name: "patch pd add additional container ",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Replicas = 5
				tc.Spec.PD.AdditionalContainers = []corev1.Container{
					{Name: "additional", Image: "test"},
				}
			},
			pdHealth: &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
				{Name: "pd1", MemberID: uint64(1), ClientUrls: []string{"http://test-pd-1.test-pd-peer.default.svc:2379"}, Health: true},
				{Name: "pd2", MemberID: uint64(2), ClientUrls: []string{"http://test-pd-2.test-pd-peer.default.svc:2379"}, Health: true},
				{Name: "pd3", MemberID: uint64(3), ClientUrls: []string{"http://test-pd-3.test-pd-peer.default.svc:2379"}, Health: false},
			}},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			errWhenGetPDHealth:         false,
			err:                        false,
			expectPDServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectPDPeerServiceFn: nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(set.Spec.Template.Spec.Containers)).To(Equal(2))
				g.Expect(set.Spec.Template.Spec.Containers[1]).To(Equal(corev1.Container{Name: "additional", Image: "test"}))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
			},
		},
	}

	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func newTidbClusterForPDMS() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       "test",
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pingcap/pd:v7.2.0",
				},
				Replicas:         1,
				StorageClassName: pointer.StringPtr("my-storage-class"),
				Mode:             "ms",
			},
			PDMS: []*v1alpha1.PDMSSpec{
				{
					Name: "tso",
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: "pingcap/pd:v7.2.0",
					},
					Replicas: 3,
				},
			},
		},
	}
}

func newFakePDMSMemberManager(ms []*v1alpha1.PDMSSpec) (*pdMSMemberManager, cache.Indexer, cache.Indexer) {
	fakeDeps := controller.NewFakeDependencies()
	podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	pdMSManager := &pdMSMemberManager{
		deps:      fakeDeps,
		suspender: suspender.NewFakeSuspender(),
	}
	if ms != nil {
		pdMSManager.curSpec = ms[0]
	}
	return pdMSManager, podIndexer, pvcIndexer
}

func TestGetNewPDMSSetForTidbCluster(t *testing.T) {
	asNonRoot := true
	privileged := true
	tests := []struct {
		name    string
		tc      v1alpha1.TidbCluster
		wantErr bool
		testSts func(sts *apps.StatefulSet)
	}{
		{
			name: "pd network is not host",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Mode: "ms",
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					TiDB: &v1alpha1.TiDBSpec{},
					PDMS: []*v1alpha1.PDMSSpec{{Name: "tso"}},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "set custom env",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Mode: "ms",
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					TiDB: &v1alpha1.TiDBSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								Env: []corev1.EnvVar{
									{
										Name: "PDMS_SESSION_SECRET",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "pdms-session-secret",
												},
												Key: "encryption_key",
											},
										},
									},
									{
										Name:  "TZ",
										Value: "ignored",
									},
								},
							},
						},
					},
				},
			},
			testSts: testContainerEnv(t, []corev1.EnvVar{
				{
					Name: "NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				},
				{
					Name:  "PEER_SERVICE_NAME",
					Value: "tc-tso-peer",
				},
				{
					Name:  "SERVICE_NAME",
					Value: "tc-tso",
				},
				{
					Name:  "SET_NAME",
					Value: "tc-tso",
				},
				{
					Name: "TZ",
				},
				{
					Name:  "CLUSTER_NAME",
					Value: "tc",
				},
				{
					Name:  "HEADLESS_SERVICE_NAME",
					Value: "tc-tso-peer",
				},
				{
					Name: "PDMS_SESSION_SECRET",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "pdms-session-secret",
							},
							Key: "encryption_key",
						},
					},
				},
			},
				v1alpha1.TSOMemberType,
			),
		},
		{
			name: "TSO additional containers",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Mode: "ms",
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					TiDB: &v1alpha1.TiDBSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								AdditionalContainers: []corev1.Container{customSideCarContainers[0]},
							},
						},
					},
				},
			},
			testSts: testAdditionalContainers(t, []corev1.Container{customSideCarContainers[0]}),
		},
		{
			name: "sysctl with no init container",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Mode: "ms",
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: &asNonRoot,
									Sysctls: []corev1.Sysctl{
										{
											Name:  "net.core.somaxconn",
											Value: "32768",
										},
										{
											Name:  "net.ipv4.tcp_syncookies",
											Value: "0",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_time",
											Value: "300",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_intvl",
											Value: "75",
										},
									},
								},
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.Template.Spec.InitContainers).Should(BeEmpty())
				g.Expect(sts.Spec.Template.Spec.SecurityContext).To(Equal(&corev1.PodSecurityContext{
					RunAsNonRoot: &asNonRoot,
					Sysctls: []corev1.Sysctl{
						{
							Name:  "net.core.somaxconn",
							Value: "32768",
						},
						{
							Name:  "net.ipv4.tcp_syncookies",
							Value: "0",
						},
						{
							Name:  "net.ipv4.tcp_keepalive_time",
							Value: "300",
						},
						{
							Name:  "net.ipv4.tcp_keepalive_intvl",
							Value: "75",
						},
					},
				}))
			},
		},
		{
			name: "sysctl with init container01",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Mode: "ms",
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								Annotations: map[string]string{
									"tidb.pingcap.com/sysctl-init": "true",
								},
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: &asNonRoot,
								},
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.Template.Spec.InitContainers).Should(BeEmpty())
				g.Expect(sts.Spec.Template.Spec.SecurityContext).To(Equal(&corev1.PodSecurityContext{
					RunAsNonRoot: &asNonRoot,
				}))
			},
		},
		{
			name: "sysctl with init container02",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Annotations: map[string]string{
								"tidb.pingcap.com/sysctl-init": "true",
							},
							PodSecurityContext: nil,
							Image:              "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{{Name: "tso"}},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.Template.Spec.InitContainers).Should(BeEmpty())
				g.Expect(sts.Spec.Template.Spec.SecurityContext).To(BeNil())
			},
		},
		{
			name: "sysctl with init container03",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								Annotations: map[string]string{
									"tidb.pingcap.com/sysctl-init": "true",
								},
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: &asNonRoot,
									Sysctls: []corev1.Sysctl{
										{
											Name:  "net.core.somaxconn",
											Value: "32768",
										},
										{
											Name:  "net.ipv4.tcp_syncookies",
											Value: "0",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_time",
											Value: "300",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_intvl",
											Value: "75",
										},
									},
								},
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.Template.Spec.InitContainers).To(Equal([]corev1.Container{
					{
						Name:  "init",
						Image: "busybox:1.26.2",
						Command: []string{
							"sh",
							"-c",
							"sysctl -w net.core.somaxconn=32768 net.ipv4.tcp_syncookies=0 net.ipv4.tcp_keepalive_time=300 net.ipv4.tcp_keepalive_intvl=75",
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
					},
				}))
				g.Expect(sts.Spec.Template.Spec.SecurityContext).To(Equal(&corev1.PodSecurityContext{
					RunAsNonRoot: &asNonRoot,
					Sysctls:      []corev1.Sysctl{},
				}))
			},
		},
		{
			name: "Specify init container resourceRequirements",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								Annotations: map[string]string{
									"tidb.pingcap.com/sysctl-init": "true",
								},
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: &asNonRoot,
									Sysctls: []corev1.Sysctl{
										{
											Name:  "net.core.somaxconn",
											Value: "32768",
										},
										{
											Name:  "net.ipv4.tcp_syncookies",
											Value: "0",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_time",
											Value: "300",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_intvl",
											Value: "75",
										},
									},
								},
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.Template.Spec.InitContainers).To(Equal([]corev1.Container{
					{
						Name:  "init",
						Image: "busybox:1.26.2",
						Command: []string{
							"sh",
							"-c",
							"sysctl -w net.core.somaxconn=32768 net.ipv4.tcp_syncookies=0 net.ipv4.tcp_keepalive_time=300 net.ipv4.tcp_keepalive_intvl=75",
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
					},
				}))
				g.Expect(sts.Spec.Template.Spec.SecurityContext).To(Equal(&corev1.PodSecurityContext{
					RunAsNonRoot: &asNonRoot,
					Sysctls:      []corev1.Sysctl{},
				}))
			},
		},
		{
			name: "sysctl without init container due to invalid annotation",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								Annotations: map[string]string{
									"tidb.pingcap.com/sysctl-init": "false",
								},
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: &asNonRoot,
									Sysctls: []corev1.Sysctl{
										{
											Name:  "net.core.somaxconn",
											Value: "32768",
										},
										{
											Name:  "net.ipv4.tcp_syncookies",
											Value: "0",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_time",
											Value: "300",
										},
										{
											Name:  "net.ipv4.tcp_keepalive_intvl",
											Value: "75",
										},
									},
								},
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.Template.Spec.InitContainers).Should(BeEmpty())
				g.Expect(sts.Spec.Template.Spec.SecurityContext).To(Equal(&corev1.PodSecurityContext{
					RunAsNonRoot: &asNonRoot,
					Sysctls: []corev1.Sysctl{
						{
							Name:  "net.core.somaxconn",
							Value: "32768",
						},
						{
							Name:  "net.ipv4.tcp_syncookies",
							Value: "0",
						},
						{
							Name:  "net.ipv4.tcp_keepalive_time",
							Value: "300",
						},
						{
							Name:  "net.ipv4.tcp_keepalive_intvl",
							Value: "75",
						},
					},
				}))
			},
		},
		{
			name: "no init container no securityContext",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tc",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{{Name: "tso"}},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.Template.Spec.InitContainers).Should(BeEmpty())
				g.Expect(sts.Spec.Template.Spec.SecurityContext).To(BeNil())
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			pdmm, _, _ := newFakePDMSMemberManager(tt.tc.Spec.PDMS)
			sts, err := pdmm.getNewPDMSStatefulSet(&tt.tc, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error %v, wantErr %v", err, tt.wantErr)
			}
			tt.testSts(sts)
		})
	}
}

func TestGetPDMSConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
	testCases := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected *corev1.ConfigMap
	}{
		{
			name: "PDMS config is nil",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{{Name: "tso"}},
				},
			},
			expected: nil,
		},
		{
			name: "basic",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiKV: &v1alpha1.TiKVSpec{},
					TiDB: &v1alpha1.TiDBSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							ComponentSpec: v1alpha1.ComponentSpec{
								ConfigUpdateStrategy: &updateStrategy,
							},
							Config: mustPDConfig(&v1alpha1.PDConfig{
								Schedule: &v1alpha1.PDScheduleConfig{
									MaxStoreDownTime:         pointer.StringPtr("5m"),
									DisableRemoveDownReplica: pointer.BoolPtr(true),
								},
								Replication: &v1alpha1.PDReplicationConfig{
									MaxReplicas:    func() *uint64 { i := uint64(5); return &i }(),
									LocationLabels: []string{"node", "rack"},
								},
							}),
						},
					},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tso",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Data: map[string]string{
					"startup-script": "",
					"config-file": `[replication]
  location-labels = ["node", "rack"]
  max-replicas = 5

[schedule]
  disable-remove-down-replica = true
  max-store-down-time = "5m"
`,
				},
			},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			pdmm, _, _ := newFakePDMSMemberManager(tt.tc.Spec.PDMS)
			if tt.tc.Spec.PDMS != nil {
				pdmm.curSpec = tt.tc.Spec.PDMS[0]
			}
			cm, err := pdmm.getPDMSConfigMap(&tt.tc)
			g.Expect(err).To(Succeed())
			if tt.expected == nil {
				g.Expect(cm).To(BeNil())
				return
			}
			// startup-script is better to be tested in e2e
			cm.Data["startup-script"] = ""
			if diff := cmp.Diff(*tt.expected, *cm); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewPdMSServiceForTidbCluster(t *testing.T) {
	tests := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected corev1.Service
	}{
		{
			name: "basic",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},

				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "tso", Type: string(corev1.ServiceTypeClusterIP)},
					},

					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{{Name: "tso"}},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tso",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
						"app.kubernetes.io/used-by":    "end-user",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       v1alpha1.DefaultPDClientPort,
							TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDClientPort)),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
					},
				},
			},
		},
		{
			name: "basic and  specify ClusterIP type,clusterIP",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "tso", Type: string(corev1.ServiceTypeClusterIP)},
					},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name:    "tso",
							Service: &v1alpha1.ServiceSpec{ClusterIP: pointer.StringPtr("172.20.10.1")},
						},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tso",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
						"app.kubernetes.io/used-by":    "end-user",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       v1alpha1.DefaultPDClientPort,
							TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDClientPort)),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
					},
				},
			},
		},
		{
			name: "basic and specify LoadBalancerIP type , LoadBalancerType",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "tso", Type: string(corev1.ServiceTypeLoadBalancer)},
					},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name:    "tso",
							Service: &v1alpha1.ServiceSpec{LoadBalancerIP: pointer.StringPtr("172.20.10.1")},
						},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tso",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
						"app.kubernetes.io/used-by":    "end-user",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					LoadBalancerIP: "172.20.10.1",
					Type:           corev1.ServiceTypeLoadBalancer,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       v1alpha1.DefaultPDClientPort,
							TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDClientPort)),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
					},
				},
			},
		},
		{
			name: "basic and specify tso service overwrite",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "tso", Type: string(corev1.ServiceTypeLoadBalancer)},
					},
					TiDB: &v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							Service: &v1alpha1.ServiceSpec{Type: corev1.ServiceTypeClusterIP,
								ClusterIP: pointer.StringPtr("172.20.10.1")},
						},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tso",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
						"app.kubernetes.io/used-by":    "end-user",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       v1alpha1.DefaultPDClientPort,
							TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDClientPort)),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
					},
				},
			},
		},
		{
			name: "basic and specify tso service portname",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Services: []v1alpha1.Service{
						{Name: "tso", Type: string(corev1.ServiceTypeLoadBalancer)},
					},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{
						TLSClient: &v1alpha1.TiDBTLSClient{
							Enabled: true,
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: "tso",
							Service: &v1alpha1.ServiceSpec{Type: corev1.ServiceTypeClusterIP,
								ClusterIP: pointer.StringPtr("172.20.10.1"),
								PortName:  pointer.StringPtr("http-tso"),
							},
						},
					},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tso",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
						"app.kubernetes.io/used-by":    "end-user",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "TidbCluster",
							Name:       "foo",
							UID:        "",
							Controller: func(b bool) *bool {
								return &b
							}(true),
							BlockOwnerDeletion: func(b bool) *bool {
								return &b
							}(true),
						},
					},
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "http-tso",
							Port:       v1alpha1.DefaultPDClientPort,
							TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDClientPort)),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
					},
				},
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			pdmm, _, _ := newFakePDMSMemberManager(tt.tc.Spec.PDMS)
			svc := pdmm.getNewPDMSService(&tt.tc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}
