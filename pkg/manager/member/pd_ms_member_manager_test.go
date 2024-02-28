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
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
		pdSvcCreated               bool
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

		pmm, _, _ := newFakePDMSMemberManager()
		fakeSetControl := pmm.deps.StatefulSetControl.(*controller.FakeStatefulSetControl)
		fakeSvcControl := pmm.deps.ServiceControl.(*controller.FakeServiceControl)
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

		err := pmm.Sync(tc)
		test.errExpectFn(g, err)
		g.Expect(tc.Spec).To(Equal(oldSpec))

		svc1, err := pmm.deps.ServiceLister.Services(ns).Get(controller.PDMSMemberName(tcName, tsoService))
		eps1, eperr := pmm.deps.EndpointLister.Endpoints(ns).Get(controller.PDMSMemberName(tcName, tsoService))
		if test.pdSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc1).NotTo(Equal(nil))
			g.Expect(eperr).NotTo(HaveOccurred())
			g.Expect(eps1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
			expectErrIsNotFound(g, eperr)
		}

		tc1, err := pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, tsoService))
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
			pdSvcCreated:               true,
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
		errWhenUpdateStatefulSet   bool
		errWhenUpdatePDService     bool
		errWhenUpdatePDPeerService bool
		statusChange               func(*apps.StatefulSet)
		err                        bool
		expectPDServiceFn          func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn        func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn        func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPDMS()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, _, _ := newFakePDMSMemberManager()
		fakePDControl := pmm.deps.PDControl.(*pdapi.FakePDControl)
		fakeSetControl := pmm.deps.StatefulSetControl.(*controller.FakeStatefulSetControl)
		fakeSvcControl := pmm.deps.ServiceControl.(*controller.FakeServiceControl)

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		pdClient := controller.NewFakePDClient(fakePDControl, tc)
		pdClient.AddReaction(pdapi.GetPDMSMembersActionType, func(action *pdapi.Action) (interface{}, error) {
			return []string{tsoService}, nil
		})

		err := pmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = pmm.deps.ServiceLister.Services(ns).Get(controller.PDMSMemberName(tcName, tsoService))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.deps.EndpointLister.Endpoints(ns).Get(controller.PDMSMemberName(tcName, tsoService))
		g.Expect(err).NotTo(HaveOccurred())

		_, err = pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, tsoService))
		g.Expect(err).NotTo(HaveOccurred())

		tc1 := tc.DeepCopy()
		test.modify(tc1)

		if test.errWhenUpdatePDService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenUpdatePDPeerService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 1)
		}
		if test.errWhenUpdateStatefulSet {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = pmm.Sync(tc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectPDServiceFn != nil {
			svc, err := pmm.deps.ServiceLister.Services(ns).Get(controller.PDMSMemberName(tcName, tsoService))
			test.expectPDServiceFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, tsoService))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc1)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDMS = []*v1alpha1.PDMSSpec{
					{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.3.0",
						},
						Name:     tsoService,
						Replicas: 5,
					},
				}
				tc.Spec.Services = []v1alpha1.Service{
					{Name: tsoService, Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err:                        false,
			expectPDServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				// g.Expect(int(*set.Spec.Replicas)).To(Equal(4))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.ScalePhase))
			},
		},
		{
			name: "patch tso add additional container ",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDMS = []*v1alpha1.PDMSSpec{
					{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.3.0",
						},
						Name:     tsoService,
						Replicas: 5,
					},
				}

				tc.Spec.PDMS[0].AdditionalContainers = []corev1.Container{
					{Name: "additional", Image: "test"},
				}
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdatePDService:     false,
			errWhenUpdatePDPeerService: false,
			err:                        false,
			expectPDServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
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

func TestPDMSMemberManagerSyncPDMSSts(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		preModify           func(cluster *v1alpha1.TidbCluster)
		modify              func(cluster *v1alpha1.TidbCluster)
		err                 bool
		expectStatefulSetFn func(*GomegaWithT, *apps.StatefulSet, error)
		expectTidbClusterFn func(*GomegaWithT, *v1alpha1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPDMS()
		if test.preModify != nil {
			test.preModify(tc)
		}
		ns := tc.Namespace
		tcName := tc.Name

		pmm, _, _ := newFakePDMSMemberManager()
		fakePDControl := pmm.deps.PDControl.(*pdapi.FakePDControl)
		pdClient := controller.NewFakePDClient(fakePDControl, tc)
		pdClient.AddReaction(pdapi.GetPDMSMembersActionType, func(action *pdapi.Action) (interface{}, error) {
			return []string{tsoService}, nil
		})

		err := pmm.Sync(tc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = pmm.deps.ServiceLister.Services(ns).Get(controller.PDMSMemberName(tcName, tsoService))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, tsoService))
		g.Expect(err).NotTo(HaveOccurred())

		test.modify(tc)
		err = pmm.syncPDMSStatefulSet(tc, tc.Spec.PDMS[0])
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, tsoService))
			test.expectStatefulSetFn(g, set, err)
			println("set.Spec.Template.Spec.Containers[0].Image", set.Spec.Template.Spec.Containers[0].Image)
		}
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc)
		}
	}
	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Image = "pingcap/pd:v7.3.0"
				tc.Spec.PDMS = []*v1alpha1.PDMSSpec{
					{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pd-test-image",
						},
						Name:     tsoService,
						Replicas: 1,
					},
				}
			},
			err: false,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("pd-test-image"))
				g.Expect(*set.Spec.Replicas).To(Equal(int32(2)))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.ScalePhase))
			},
		},
		{
			name: "emptyImage",
			modify: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Image = "pingcap/pd:v7.3.0"
				tc.Spec.PDMS = []*v1alpha1.PDMSSpec{
					{
						ComponentSpec: v1alpha1.ComponentSpec{},
						Name:          tsoService,
						Replicas:      1,
					},
				}
			},
			err: false,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("pingcap/pd:v7.3.0"))
				g.Expect(*set.Spec.Replicas).To(Equal(int32(2)))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.PDMS[tsoService].Phase).To(Equal(v1alpha1.ScalePhase))
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
					Image: "pingcap/pd:v7.3.0",
				},
				Replicas:         1,
				StorageClassName: pointer.StringPtr("my-storage-class"),
				Mode:             "ms",
			},
			PDMS: []*v1alpha1.PDMSSpec{
				{
					Name: tsoService,
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: "pingcap/pd:v7.3.0",
					},
					Replicas: 3,
				},
			},
		},
	}
}

func TestGetNewPDMSHeadlessServiceForTidbCluster(t *testing.T) {
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
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-tso-peer",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "tso",
						"app.kubernetes.io/used-by":    "peer",
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
					ClusterIP: "None",
					Ports: []corev1.ServicePort{
						{
							Name:       "tcp-peer-2380",
							Port:       v1alpha1.DefaultPDPeerPort,
							TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDPeerPort)),
							Protocol:   corev1.ProtocolTCP,
						},
						{
							Name:       "tcp-peer-2379",
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
					PublishNotReadyAddresses: true,
				},
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewPDMSHeadlessService(&tt.tc, tsoService)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}

func newFakePDMSMemberManager() (*pdMSMemberManager, cache.Indexer, cache.Indexer) {
	fakeDeps := controller.NewFakeDependencies()
	podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	pdMSManager := &pdMSMemberManager{
		deps:              fakeDeps,
		scaler:            NewFakePDMSScaler(),
		upgrader:          NewFakePDMSUpgrader(),
		suspender:         suspender.NewFakeSuspender(),
		podVolumeModifier: &volumes.FakePodVolumeModifier{},
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
					PDMS: []*v1alpha1.PDMSSpec{{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.3.0",
						},
						Name: tsoService}},
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
							Image: "pingcap/pd:v7.3.0",
						},
					},
					TiKV: &v1alpha1.TiKVSpec{},
					TiDB: &v1alpha1.TiDBSpec{},
					PDMS: []*v1alpha1.PDMSSpec{
						{
							Name: tsoService,
							ComponentSpec: v1alpha1.ComponentSpec{
								Image: "pingcap/pd:v7.3.0",
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
				v1alpha1.PDMSTSOMemberType,
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
							Name: tsoService,
							ComponentSpec: v1alpha1.ComponentSpec{
								Image:                "pingcap/pd:v7.3.0",
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
							Name: tsoService,
							ComponentSpec: v1alpha1.ComponentSpec{
								Image: "pingcap/pd:v7.3.0",
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
							Name: tsoService,
							ComponentSpec: v1alpha1.ComponentSpec{
								Image: "pingcap/pd:v7.3.0",
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
					PDMS: []*v1alpha1.PDMSSpec{{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.3.0",
						},
						Name: tsoService}},
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
							Name: tsoService,
							ComponentSpec: v1alpha1.ComponentSpec{
								Image: "pingcap/pd:v7.3.0",
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
							Name: tsoService,
							ComponentSpec: v1alpha1.ComponentSpec{
								Image: "pingcap/pd:v7.3.0",
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
							Name: tsoService,
							ComponentSpec: v1alpha1.ComponentSpec{
								Image: "pingcap/pd:v7.3.0",
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
					PDMS: []*v1alpha1.PDMSSpec{{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.3.0",
						},
						Name: tsoService}},
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
			pdmm, _, _ := newFakePDMSMemberManager()
			sts, err := pdmm.getNewPDMSStatefulSet(&tt.tc, nil, tt.tc.Spec.PDMS[0])
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
					PDMS: []*v1alpha1.PDMSSpec{{Name: tsoService}},
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
					"config-file":    "",
				},
			},
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
							Name: tsoService,
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
		{
			name: "enable tls when config is nil",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
					PD: &v1alpha1.PDSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Image: "pingcap/pd:v7.2.0",
						},
						Mode: "ms",
					},
					TiDB: &v1alpha1.TiDBSpec{},
					TiKV: &v1alpha1.TiKVSpec{},
					PDMS: []*v1alpha1.PDMSSpec{{Name: tsoService}},
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
					"config-file": `[security]
  cacert-path = "/var/lib/pd-tls/ca.crt"
  cert-path = "/var/lib/pd-tls/tls.crt"
  key-path = "/var/lib/pd-tls/tls.key"
`,
				},
			},
		},
		{
			name: "basic config with tls",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
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
							Name: tsoService,
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

[security]
  cacert-path = "/var/lib/pd-tls/ca.crt"
  cert-path = "/var/lib/pd-tls/tls.crt"
  key-path = "/var/lib/pd-tls/tls.key"
`,
				},
			},
		},
	}

	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			pdmm, _, _ := newFakePDMSMemberManager()
			cm, err := pdmm.getPDMSConfigMap(&tt.tc, tt.tc.Spec.PDMS[0])
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
						{Name: tsoService, Type: string(corev1.ServiceTypeClusterIP)},
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
					PDMS: []*v1alpha1.PDMSSpec{{Name: tsoService}},
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
						{Name: tsoService, Type: string(corev1.ServiceTypeClusterIP)},
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
							Name:    tsoService,
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
						{Name: tsoService, Type: string(corev1.ServiceTypeLoadBalancer)},
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
							Name:    tsoService,
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
						{Name: tsoService, Type: string(corev1.ServiceTypeLoadBalancer)},
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
							Name: tsoService,
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
						{Name: tsoService, Type: string(corev1.ServiceTypeLoadBalancer)},
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
							Name: tsoService,
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
			pdmm, _, _ := newFakePDMSMemberManager()
			svc := pdmm.getNewPDMSService(&tt.tc, tt.tc.Spec.PDMS[0])
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}
