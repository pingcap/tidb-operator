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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestPumpMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)

	type result struct {
		sync   error
		svc    *corev1.Service
		getSvc error
		set    *appsv1.StatefulSet
		getSet error
		cm     *corev1.ConfigMap
		getCm  error
	}

	type testcase struct {
		name           string
		prepare        func(cluster *v1alpha1.TidbCluster)
		errOnCreateSet bool
		errOnCreateCm  bool
		errOnCreateSvc bool
		expectFn       func(*GomegaWithT, *result)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPump()
		ns := tc.Namespace
		tcName := tc.Name
		if test.prepare != nil {
			test.prepare(tc)
		}

		pmm, ctls, _ := newFakePumpMemberManager()

		if test.errOnCreateSet {
			ctls.set.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnCreateSvc {
			ctls.svc.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnCreateCm {
			ctls.generic.SetCreateOrUpdateError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		syncErr := pmm.Sync(tc)
		svc, getSvcErr := pmm.deps.ServiceLister.Services(ns).Get(controller.PumpPeerMemberName(tcName))
		set, getStsErr := pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PumpMemberName(tcName))
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: controller.PumpMemberName(tcName)}}
		key, err := client.ObjectKeyFromObject(cm)
		g.Expect(err).To(Succeed())
		getCmErr := ctls.generic.FakeCli.Get(context.TODO(), key, cm)
		result := result{syncErr, svc, getSvcErr, set, getStsErr, cm, getCmErr}
		test.expectFn(g, &result)
	}

	tests := []*testcase{
		{
			name:           "basic",
			prepare:        nil,
			errOnCreateSet: false,
			errOnCreateCm:  false,
			errOnCreateSvc: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).To(Succeed())
				g.Expect(r.getCm).To(Succeed())
				g.Expect(r.getSet).To(Succeed())
				g.Expect(r.getSvc).To(Succeed())
			},
		},
		{
			name: "do not sync if pum spec is nil",
			prepare: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.Pump = nil
			},
			errOnCreateSet: false,
			errOnCreateCm:  false,
			errOnCreateSvc: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).To(Succeed())
				g.Expect(r.getCm).NotTo(Succeed())
				g.Expect(r.getSet).NotTo(Succeed())
				g.Expect(r.getSvc).NotTo(Succeed())
			},
		},
		{
			name:           "error when create pump statefulset",
			prepare:        nil,
			errOnCreateSet: true,
			errOnCreateCm:  false,
			errOnCreateSvc: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.getSet).NotTo(Succeed())
				g.Expect(r.getCm).To(Succeed())
				g.Expect(r.getSvc).To(Succeed())
			},
		},
		{
			name:           "error when create pump peer service",
			prepare:        nil,
			errOnCreateSet: false,
			errOnCreateCm:  false,
			errOnCreateSvc: true,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.getSet).NotTo(Succeed())
				g.Expect(r.getCm).NotTo(Succeed())
				g.Expect(r.getSvc).NotTo(Succeed())
			},
		},
		{
			name:           "error when create pump configmap",
			prepare:        nil,
			errOnCreateSet: false,
			errOnCreateCm:  true,
			errOnCreateSvc: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.getSet).NotTo(Succeed())
				g.Expect(r.getCm).NotTo(Succeed())
				g.Expect(r.getSvc).To(Succeed())
			},
		},
	}

	for _, tt := range tests {
		testFn(tt, t)
	}
}

func TestPumpMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)

	type result struct {
		sync   error
		oldSvc *corev1.Service
		svc    *corev1.Service
		getSvc error
		oldSet *appsv1.StatefulSet
		set    *appsv1.StatefulSet
		getSet error
		oldCm  *corev1.ConfigMap
		cm     *corev1.ConfigMap
		getCm  error
	}
	type testcase struct {
		name           string
		prepare        func(*v1alpha1.TidbCluster, *pumpFakeIndexers)
		errOnUpdateSet bool
		errOnUpdateCm  bool
		errOnUpdateSvc bool
		expectFn       func(*GomegaWithT, *result)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForPump()
		ns := tc.Namespace
		tcName := tc.Name

		pmm, ctls, indexers := newFakePumpMemberManager()

		if test.errOnUpdateSet {
			ctls.set.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnUpdateSvc {
			ctls.svc.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnUpdateCm {
			ctls.generic.SetCreateOrUpdateError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		oldCm, err := getNewPumpConfigMap(tc)
		g.Expect(err).To(Succeed())
		oldSvc := getNewPumpHeadlessService(tc)
		oldSvc.Spec.Ports[0].Port = 8888
		oldSet, err := getNewPumpStatefulSet(tc, oldCm)
		g.Expect(err).To(Succeed())

		g.Expect(indexers.set.Add(oldSet)).To(Succeed())
		g.Expect(indexers.svc.Add(oldSvc)).To(Succeed())

		g.Expect(ctls.generic.AddObject(oldCm)).To(Succeed())

		if test.prepare != nil {
			test.prepare(tc, indexers)
		}

		syncErr := pmm.Sync(tc)
		svc, getSvcErr := pmm.deps.ServiceLister.Services(ns).Get(controller.PumpPeerMemberName(tcName))
		set, getStsErr := pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PumpMemberName(tcName))
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: controller.PumpMemberName(tcName)}}
		key, err := client.ObjectKeyFromObject(cm)
		g.Expect(err).To(Succeed())
		getCmErr := ctls.generic.FakeCli.Get(context.TODO(), key, cm)
		result := result{syncErr, oldSvc, svc, getSvcErr, oldSet, set, getStsErr, oldCm, cm, getCmErr}
		test.expectFn(g, &result)
	}

	tests := []*testcase{
		{
			name: "basic",
			prepare: func(tc *v1alpha1.TidbCluster, _ *pumpFakeIndexers) {
				tc.Spec.Pump.Config = config.New(map[string]interface{}{
					"gc": 6,
					"storage": map[string]interface{}{
						"stop-write-at-available-space": "10Gi",
					},
				})
				tc.Spec.Pump.Replicas = 5
			},
			errOnUpdateCm:  false,
			errOnUpdateSvc: false,
			errOnUpdateSet: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).To(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).NotTo(Equal(int32(8888)))
				g.Expect(r.cm.Data["pump-config"]).To(ContainSubstring("stop-write-at-available-space"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(5)))
			},
		},
		{
			name: "error on update configmap",
			prepare: func(tc *v1alpha1.TidbCluster, _ *pumpFakeIndexers) {
				tc.Spec.Pump.Config = config.New(map[string]interface{}{
					"gc": 6,
					"storage": map[string]interface{}{
						"stop-write-at-available-space": "10Gi",
					},
				})
				tc.Spec.Pump.Replicas = 5
			},
			errOnUpdateCm:  true,
			errOnUpdateSvc: false,
			errOnUpdateSet: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).NotTo(Equal(int32(8888)))
				g.Expect(r.cm.Data["pump-config"]).NotTo(ContainSubstring("stop-write-at-available-space"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(3)))
			},
		},
		{
			name: "error on update service",
			prepare: func(tc *v1alpha1.TidbCluster, _ *pumpFakeIndexers) {
				tc.Spec.Pump.Config = config.New(map[string]interface{}{
					"gc": 6,
					"storage": map[string]interface{}{
						"stop-write-at-available-space": "10Gi",
					},
				})
				tc.Spec.Pump.Replicas = 5
			},
			errOnUpdateCm:  false,
			errOnUpdateSvc: true,
			errOnUpdateSet: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).To(Equal(int32(8888)))
				g.Expect(r.cm.Data["pump-config"]).NotTo(ContainSubstring("stop-write-at-available-space"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(3)))
			},
		},
		{
			name: "error on update statefulset",
			prepare: func(tc *v1alpha1.TidbCluster, _ *pumpFakeIndexers) {
				tc.Spec.Pump.Config = config.New(map[string]interface{}{
					"gc": 6,
					"storage": map[string]interface{}{
						"stop-write-at-available-space": "10Gi",
					},
				})
				tc.Spec.Pump.Replicas = 5
			},
			errOnUpdateCm:  false,
			errOnUpdateSvc: false,
			errOnUpdateSet: true,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).NotTo(Equal(int32(8888)))
				g.Expect(r.cm.Data["pump-config"]).To(ContainSubstring("stop-write-at-available-space"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(3)))
			},
		},
	}

	for _, tt := range tests {
		testFn(tt, t)
	}
}

func TestSyncConfigUpdate(t *testing.T) {
	g := NewGomegaWithT(t)

	type result struct {
		sync   error
		oldSet *appsv1.StatefulSet
		set    *appsv1.StatefulSet
		getSet error
		oldCm  *corev1.ConfigMap
		cms    []corev1.ConfigMap
		listCm error
	}
	type testcase struct {
		name     string
		prepare  func(*v1alpha1.TidbCluster, *pumpFakeIndexers)
		expectFn func(*GomegaWithT, *result)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForPump()
		ns := tc.Namespace
		tcName := tc.Name
		updateStrategy := v1alpha1.ConfigUpdateStrategyRollingUpdate
		tc.Spec.Pump.ConfigUpdateStrategy = &updateStrategy

		pmm, controls, indexers := newFakePumpMemberManager()

		oldCm, err := getNewPumpConfigMap(tc)
		g.Expect(err).To(Succeed())
		oldSvc := getNewPumpHeadlessService(tc)
		oldSvc.Spec.Ports[0].Port = 8888
		oldSet, err := getNewPumpStatefulSet(tc, oldCm)
		g.Expect(err).To(Succeed())

		g.Expect(indexers.set.Add(oldSet)).To(Succeed())
		g.Expect(indexers.svc.Add(oldSvc)).To(Succeed())
		g.Expect(controls.generic.AddObject(oldCm)).To(Succeed())

		if test.prepare != nil {
			test.prepare(tc, indexers)
		}

		syncErr := pmm.Sync(tc)
		set, getStsErr := pmm.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PumpMemberName(tcName))
		cmList := &corev1.ConfigMapList{}
		g.Expect(err).To(Succeed())
		listCmErr := controls.generic.FakeCli.List(context.TODO(), cmList)
		result := result{syncErr, oldSet, set, getStsErr, oldCm, cmList.Items, listCmErr}
		test.expectFn(g, &result)
	}

	tests := []*testcase{
		{
			name: "basic",
			prepare: func(tc *v1alpha1.TidbCluster, _ *pumpFakeIndexers) {
				tc.Spec.Pump.Config = config.New(map[string]interface{}{
					"gc": 6,
					"storage": map[string]interface{}{
						"stop-write-at-available-space": "10Gi",
					},
				})
			},
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).To(Succeed())
				g.Expect(r.listCm).To(Succeed())
				g.Expect(r.cms).To(HaveLen(2))
				g.Expect(r.getSet).To(Succeed())
				using := FindConfigMapVolume(&r.set.Spec.Template.Spec, func(name string) bool {
					return strings.HasPrefix(name, controller.PumpMemberName("test"))
				})
				g.Expect(using).NotTo(BeEmpty())
				var usingCm *corev1.ConfigMap
				for _, cm := range r.cms {
					if cm.Name == using {
						usingCm = &cm
					}
				}
				g.Expect(usingCm).NotTo(BeNil(), "The configmap used by statefulset must be created")
				g.Expect(usingCm.Data["pump-config"]).To(ContainSubstring("stop-write-at-available-space"),
					"The configmap used by statefulset should be the latest one")
			},
		},
	}

	for _, tt := range tests {
		testFn(tt, t)
	}
}

type pumpFakeIndexers struct {
	tc  cache.Indexer
	svc cache.Indexer
	set cache.Indexer
}

type pumpFakeControls struct {
	svc     *controller.FakeServiceControl
	set     *controller.FakeStatefulSetControl
	generic *controller.FakeGenericControl
}

func newFakePumpMemberManager() (*pumpMemberManager, *pumpFakeControls, *pumpFakeIndexers) {
	fakeDeps := controller.NewFakeDependencies()
	pmm := &pumpMemberManager{deps: fakeDeps}
	controls := &pumpFakeControls{
		svc:     fakeDeps.ServiceControl.(*controller.FakeServiceControl),
		set:     fakeDeps.StatefulSetControl.(*controller.FakeStatefulSetControl),
		generic: controller.NewFakeGenericControl(),
	}
	indexers := &pumpFakeIndexers{
		tc:  fakeDeps.TiDBClusterInformer.Informer().GetIndexer(),
		svc: fakeDeps.KubeInformerFactory.Core().V1().Services().Informer().GetIndexer(),
		set: fakeDeps.KubeInformerFactory.Apps().V1().StatefulSets().Informer().GetIndexer(),
	}
	return pmm, controls, indexers
}

func newTidbClusterForPump() *v1alpha1.TidbCluster {
	updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         1,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiKV: &v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
				Replicas:         1,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiDB: &v1alpha1.TiDBSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tidb-test-image",
				},
				Replicas: 1,
			},
			Pump: &v1alpha1.PumpSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image:                "pump-test-image",
					ConfigUpdateStrategy: &updateStrategy,
				},
				Config: config.New(map[string]interface{}{
					"gc": 7,
				}),
				Replicas: 3,
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("1"),
						corev1.ResourceMemory:  resource.MustParse("2Gi"),
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				},
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
		},
	}
}

func TestGetNewPumpHeadlessService(t *testing.T) {
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
					Pump: &v1alpha1.PumpSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pump",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pump",
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
							Name:       "pump",
							Port:       8250,
							TargetPort: intstr.FromInt(8250),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pump",
					},
					PublishNotReadyAddresses: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewPumpHeadlessService(&tt.tc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewPumpConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)

	updateStrategy := v1alpha1.ConfigUpdateStrategyInPlace
	tests := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected corev1.ConfigMap
	}{
		{
			name: "empty config",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Pump: &v1alpha1.PumpSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							ConfigUpdateStrategy: &updateStrategy,
						},
						Config: nil,
					},
				},
			},
			expected: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pump",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pump",
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
					"pump-config": "",
				},
			},
		},
		{
			name: "inplace update",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					Pump: &v1alpha1.PumpSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							ConfigUpdateStrategy: &updateStrategy,
						},
						Config: config.New(map[string]interface{}{
							"gc": 7,
							"storage": map[string]interface{}{
								"sync-log": "true",
							},
						}),
					},
				},
			},
			expected: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-pump",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "pump",
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
					"pump-config": `gc = 7

[storage]
  sync-log = "true"
`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := getNewPumpConfigMap(&tt.tc)
			g.Expect(err).To(Succeed())
			if diff := cmp.Diff(tt.expected, *cm); diff != "" {
				t.Errorf("unexpected ConfigMap (-want, +got): %s", diff)
			}
		})
	}
}

// TODO: add ut for getPumpStatefulSet
func TestSyncTiDBClusterStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name     string
		updateTC func(*appsv1.StatefulSet)
		// TODO check work as expected
		// `upgradingFn` is unused
		// nolint(structcheck)
		upgradingFn func(corelisters.PodLister, *appsv1.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
		errExpectFn func(*GomegaWithT, error)
		tcExpectFn  func(*GomegaWithT, *v1alpha1.TidbCluster)
	}
	status := appsv1.StatefulSetStatus{
		Replicas: int32(3),
	}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPump()

		set := &appsv1.StatefulSet{
			Status: status,
		}
		if test.updateTC != nil {
			test.updateTC(set)
		}
		pmm, _, _ := newFakePumpMemberManager()

		err := pmm.syncTiDBClusterStatus(tc, set)

		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
		if test.tcExpectFn != nil {
			test.tcExpectFn(g, tc)
		}
	}
	tests := []testcase{
		{
			name: "statefulset is upgrading",
			updateTC: func(set *appsv1.StatefulSet) {
				set.Status.CurrentRevision = "pump-v1"
				set.Status.UpdateRevision = "pump-v2"
			},
			upgradingFn: func(lister corelisters.PodLister, set *appsv1.StatefulSet, cluster *v1alpha1.TidbCluster) (bool, error) {
				return true, nil
			},
			errExpectFn: nil,
			tcExpectFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster) {
				g.Expect(tc.Status.Pump.StatefulSet.Replicas).To(Equal(int32(3)))
				g.Expect(tc.Status.Pump.Phase).To(Equal(v1alpha1.UpgradePhase))
			},
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}
