// Copyright 2020 PingCAP, Inc.
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

	"github.com/pingcap/tidb-operator/pkg/label"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkerMemberManagerSyncCreate(t *testing.T) {
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
		prepare        func(cluster *v1alpha1.DMCluster)
		errOnCreateSet bool
		errOnCreateCm  bool
		errOnCreateSvc bool
		expectFn       func(*GomegaWithT, *result)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		dc := newDMClusterForWorker()
		ns := dc.Namespace
		dcName := dc.Name
		if test.prepare != nil {
			test.prepare(dc)
		}

		wmm, ctls, _, _ := newFakeWorkerMemberManager()

		if test.errOnCreateSet {
			ctls.set.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnCreateSvc {
			ctls.svc.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnCreateCm {
			ctls.generic.SetCreateOrUpdateError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		syncErr := wmm.SyncDM(dc)
		svc, getSvcErr := wmm.svcLister.Services(ns).Get(controller.DMWorkerPeerMemberName(dcName))
		set, getStsErr := wmm.setLister.StatefulSets(ns).Get(controller.DMWorkerMemberName(dcName))

		cmName := controller.DMWorkerMemberName(dcName)
		if dc.Spec.Worker != nil {
			cmGen, err := getWorkerConfigMap(dc)
			g.Expect(err).To(Succeed())
			cmName = cmGen.Name
			g.Expect(strings.HasPrefix(cmName, controller.DMWorkerMemberName(dcName))).To(BeTrue())
		}
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: cmName}}
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
			name: "do not sync if dm-worker spec is nil",
			prepare: func(dc *v1alpha1.DMCluster) {
				dc.Spec.Worker = nil
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
			name:           "error when create dm-worker statefulset",
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
			name:           "error when create dm-worker peer service",
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
			name:           "error when create dm-worker configmap",
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

func TestWorkerMemberManagerSyncUpdate(t *testing.T) {
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

		triggerDeleteWorker bool
	}
	type testcase struct {
		name           string
		prepare        func(*v1alpha1.DMCluster, *workerFakeIndexers)
		errOnUpdateSet bool
		errOnUpdateCm  bool
		errOnUpdateSvc bool
		expectFn       func(*GomegaWithT, *result)
		workerInfos    []*dmapi.WorkersInfo
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		dc := newDMClusterForWorker()
		ns := dc.Namespace
		dcName := dc.Name
		triggerDeleteWorker := false

		mmm, ctls, indexers, fakeMasterControl := newFakeWorkerMemberManager()

		masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)
		masterClient.AddReaction(dmapi.GetWorkersActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.workerInfos, nil
		})
		masterClient.AddReaction(dmapi.DeleteWorkerActionType, func(action *dmapi.Action) (interface{}, error) {
			triggerDeleteWorker = true
			return nil, nil
		})

		if test.errOnUpdateSet {
			ctls.set.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnUpdateSvc {
			ctls.svc.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errOnUpdateCm {
			ctls.generic.SetCreateOrUpdateError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		oldCm, err := getWorkerConfigMap(dc)
		g.Expect(err).To(Succeed())
		oldSvc := getNewWorkerHeadlessServiceForDMCluster(dc)
		oldSvc.Spec.Ports[0].Port = 8888
		oldSet, err := getNewWorkerSetForDMCluster(dc, oldCm)
		g.Expect(err).To(Succeed())

		g.Expect(indexers.set.Add(oldSet)).To(Succeed())
		g.Expect(indexers.svc.Add(oldSvc)).To(Succeed())

		g.Expect(ctls.generic.AddObject(oldCm)).To(Succeed())

		if test.prepare != nil {
			test.prepare(dc, indexers)
		}

		syncErr := mmm.SyncDM(dc)
		svc, getSvcErr := mmm.svcLister.Services(ns).Get(controller.DMWorkerPeerMemberName(dcName))
		set, getStsErr := mmm.setLister.StatefulSets(ns).Get(controller.DMWorkerMemberName(dcName))

		cmName := controller.DMWorkerMemberName(dcName)
		if dc.Spec.Worker != nil {
			cmGen, err := getWorkerConfigMap(dc)
			g.Expect(err).To(Succeed())
			cmName = cmGen.Name
			g.Expect(strings.HasPrefix(cmName, controller.DMWorkerMemberName(dcName))).To(BeTrue())
		}
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: cmName}}
		key, err := client.ObjectKeyFromObject(cm)
		g.Expect(err).To(Succeed())
		getCmErr := ctls.generic.FakeCli.Get(context.TODO(), key, cm)
		result := result{syncErr, oldSvc, svc, getSvcErr, oldSet, set, getStsErr, oldCm, cm, getCmErr, triggerDeleteWorker}
		test.expectFn(g, &result)
	}

	tests := []*testcase{
		{
			name: "basic",
			prepare: func(dc *v1alpha1.DMCluster, _ *workerFakeIndexers) {
				dc.Spec.Worker.Config = &v1alpha1.WorkerConfig{
					LogLevel:     pointer.StringPtr("info"),
					KeepAliveTTL: pointer.Int64Ptr(25),
				}
				dc.Spec.Worker.Replicas = 4
			},
			errOnUpdateCm:  false,
			errOnUpdateSvc: false,
			errOnUpdateSet: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).To(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).NotTo(Equal(int32(8888)))
				g.Expect(r.cm.Data["config-file"]).To(ContainSubstring("keepalive-ttl"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(4)))
				g.Expect(r.triggerDeleteWorker).To(BeFalse())
			},
			workerInfos: nil,
		},
		{
			name: "error on update configmap",
			prepare: func(dc *v1alpha1.DMCluster, _ *workerFakeIndexers) {
				dc.Spec.Worker.Config = &v1alpha1.WorkerConfig{
					LogLevel:     pointer.StringPtr("info"),
					KeepAliveTTL: pointer.Int64Ptr(25),
				}
				dc.Spec.Worker.Replicas = 4
			},
			errOnUpdateCm:  true,
			errOnUpdateSvc: false,
			errOnUpdateSet: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).NotTo(Equal(int32(8888)))
				g.Expect(r.cm.Data["config-file"]).NotTo(ContainSubstring("keepalive-ttl"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(3)))
				g.Expect(r.triggerDeleteWorker).To(BeFalse())
			},
			workerInfos: []*dmapi.WorkersInfo{
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 0), Addr: "http://worker0:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 1), Addr: "http://worker1:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 2), Addr: "http://worker2:8262", Stage: v1alpha1.DMWorkerStateFree},
			},
		},
		{
			name: "error on update service",
			prepare: func(dc *v1alpha1.DMCluster, _ *workerFakeIndexers) {
				dc.Spec.Worker.Config = &v1alpha1.WorkerConfig{
					LogLevel:     pointer.StringPtr("info"),
					KeepAliveTTL: pointer.Int64Ptr(25),
				}
				dc.Spec.Worker.Replicas = 4
			},
			errOnUpdateCm:  false,
			errOnUpdateSvc: true,
			errOnUpdateSet: false,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).To(Equal(int32(8888)))
				g.Expect(r.cm.Data["config-file"]).NotTo(ContainSubstring("keepalive-ttl"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(3)))
				g.Expect(r.triggerDeleteWorker).To(BeFalse())
			},
			workerInfos: []*dmapi.WorkersInfo{
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 0), Addr: "http://worker0:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 1), Addr: "http://worker1:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 2), Addr: "http://worker2:8262", Stage: v1alpha1.DMWorkerStateFree},
			},
		},
		{
			name: "error on update statefulset",
			prepare: func(dc *v1alpha1.DMCluster, _ *workerFakeIndexers) {
				dc.Spec.Worker.Config = &v1alpha1.WorkerConfig{
					LogLevel:     pointer.StringPtr("info"),
					KeepAliveTTL: pointer.Int64Ptr(25),
				}
				dc.Spec.Worker.Replicas = 4
			},
			errOnUpdateCm:  false,
			errOnUpdateSvc: false,
			errOnUpdateSet: true,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).NotTo(Equal(int32(8888)))
				g.Expect(r.cm.Data["config-file"]).To(ContainSubstring("keepalive-ttl"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(3)))
				g.Expect(r.triggerDeleteWorker).To(BeFalse())
			},
			workerInfos: []*dmapi.WorkersInfo{
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 0), Addr: "http://worker0:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 1), Addr: "http://worker1:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 2), Addr: "http://worker2:8262", Stage: v1alpha1.DMWorkerStateFree},
			},
		},
		{
			name: "offline scaled dm-worker",
			prepare: func(dc *v1alpha1.DMCluster, _ *workerFakeIndexers) {
				dc.Spec.Worker.Config = &v1alpha1.WorkerConfig{
					LogLevel:     pointer.StringPtr("info"),
					KeepAliveTTL: pointer.Int64Ptr(25),
				}
				dc.Spec.Worker.Replicas = 3
			},
			errOnUpdateCm:  false,
			errOnUpdateSvc: false,
			errOnUpdateSet: true,
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).NotTo(Succeed())
				g.Expect(r.svc.Spec.Ports[0].Port).NotTo(Equal(int32(8888)))
				g.Expect(r.cm.Data["config-file"]).To(ContainSubstring("keepalive-ttl"))
				g.Expect(*r.set.Spec.Replicas).To(Equal(int32(3)))
				g.Expect(r.triggerDeleteWorker).To(BeTrue())
			},
			workerInfos: []*dmapi.WorkersInfo{
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 0), Addr: "http://worker0:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 1), Addr: "http://worker1:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 2), Addr: "http://worker2:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: ordinalPodName(v1alpha1.DMWorkerMemberType, "test", 3), Addr: "http://worker3:8262", Stage: v1alpha1.DMWorkerStateOffline},
			},
		},
	}

	for _, tt := range tests {
		testFn(tt, t)
	}
}

func TestWorkerMemberManagerWorkerStatefulSetIsUpgrading(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		setUpdate       func(*appsv1.StatefulSet)
		hasPod          bool
		updatePod       func(*corev1.Pod)
		errExpectFn     func(*GomegaWithT, error)
		expectUpgrading bool
	}
	testFn := func(test *testcase, t *testing.T) {
		mmm, _, indexers, _ := newFakeWorkerMemberManager()
		dc := newDMClusterForWorker()
		dc.Status.Worker.StatefulSet = &appsv1.StatefulSetStatus{
			UpdateRevision: "v3",
		}

		set := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
		}
		if test.setUpdate != nil {
			test.setUpdate(set)
		}

		if test.hasPod {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ordinalPodName(v1alpha1.DMWorkerMemberType, dc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.NewDM().Instance(dc.GetInstanceName()).DMWorker().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			indexers.pod.Add(pod)
		}
		b, err := mmm.workerStatefulSetIsUpgrading(set, dc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
		if test.expectUpgrading {
			g.Expect(b).To(BeTrue())
		} else {
			g.Expect(b).NotTo(BeTrue())
		}
	}
	tests := []testcase{
		{
			name: "stateful set is upgrading",
			setUpdate: func(set *appsv1.StatefulSet) {
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v2"
				set.Status.ObservedGeneration = 1000
			},
			hasPod:          false,
			updatePod:       nil,
			errExpectFn:     nil,
			expectUpgrading: true,
		},
		{
			name:            "pod don't have revision hash",
			setUpdate:       nil,
			hasPod:          true,
			updatePod:       nil,
			errExpectFn:     nil,
			expectUpgrading: false,
		},
		{
			name:      "pod have revision hash, not equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[appsv1.ControllerRevisionHashLabelKey] = "v2"
			},
			errExpectFn:     nil,
			expectUpgrading: true,
		},
		{
			name:      "pod have revision hash, equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[appsv1.ControllerRevisionHashLabelKey] = "v3"
			},
			errExpectFn:     nil,
			expectUpgrading: false,
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func TestWorkerMemberManagerUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.DMCluster)
		workerInfos         []*dmapi.WorkersInfo
		err                 bool
		statusChange        func(*appsv1.StatefulSet)
		expectStatefulSetFn func(*GomegaWithT, *appsv1.StatefulSet, error)
		expectDMClusterFn   func(*GomegaWithT, *v1alpha1.DMCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		dc := newDMClusterForWorker()
		ns := dc.Namespace
		dcName := dc.Name

		wmm, ctls, _, fakeMasterControl := newFakeWorkerMemberManager()
		masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)
		masterClient.AddReaction(dmapi.GetWorkersActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.workerInfos, nil
		})

		ctls.set.SetStatusChange(test.statusChange)

		err := wmm.SyncDM(dc)
		g.Expect(err).To(Succeed())

		_, err = wmm.svcLister.Services(ns).Get(controller.DMWorkerPeerMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = wmm.setLister.StatefulSets(ns).Get(controller.DMWorkerMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())

		dc1 := dc.DeepCopy()
		test.modify(dc1)

		err = wmm.SyncDM(dc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := wmm.setLister.StatefulSets(ns).Get(controller.DMWorkerMemberName(dcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectDMClusterFn != nil {
			test.expectDMClusterFn(g, dc1)
		}
	}
	tests := []testcase{
		{
			name: "upgrade successful",
			modify: func(cluster *v1alpha1.DMCluster) {
				cluster.Spec.Worker.BaseImage = "dm-test-image-2"
			},
			workerInfos: []*dmapi.WorkersInfo{
				{Name: "worker1", Addr: "http://worker1:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: "worker2", Addr: "http://worker2:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: "worker3", Addr: "http://worker3:8262", Stage: v1alpha1.DMWorkerStateBound, Source: "mysql1"},
			},
			err: false,
			statusChange: func(set *appsv1.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "dm-worker-1"
				set.Status.UpdateRevision = "dm-worker-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *appsv1.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("dm-test-image-2:v2.0.0-rc.2"))
			},
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(len(dc.Status.Worker.Members)).To(Equal(3))
				g.Expect(dc.Status.Worker.Members["worker1"].Stage).To(Equal(v1alpha1.DMWorkerStateFree))
				g.Expect(dc.Status.Worker.Members["worker2"].Stage).To(Equal(v1alpha1.DMWorkerStateFree))
				g.Expect(dc.Status.Worker.Members["worker3"].Stage).To(Equal(v1alpha1.DMWorkerStateBound))
			},
		},
	}
	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestWorkerSyncConfigUpdate(t *testing.T) {
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
		name        string
		prepare     func(*v1alpha1.DMCluster, *workerFakeIndexers)
		expectFn    func(*GomegaWithT, *result)
		workerInfos []*dmapi.WorkersInfo
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		dc := newDMClusterForWorker()
		ns := dc.Namespace
		dcName := dc.Name

		mmm, controls, indexers, fakeMasterControl := newFakeWorkerMemberManager()
		masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)
		masterClient.AddReaction(dmapi.GetWorkersActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.workerInfos, nil
		})

		oldCm, err := getWorkerConfigMap(dc)
		g.Expect(err).To(Succeed())
		oldSvc := getNewWorkerHeadlessServiceForDMCluster(dc)
		oldSvc.Spec.Ports[0].Port = 8888
		oldSet, err := getNewWorkerSetForDMCluster(dc, oldCm)
		g.Expect(err).To(Succeed())

		g.Expect(indexers.set.Add(oldSet)).To(Succeed())
		g.Expect(indexers.svc.Add(oldSvc)).To(Succeed())
		g.Expect(controls.generic.AddObject(oldCm)).To(Succeed())

		if test.prepare != nil {
			test.prepare(dc, indexers)
		}

		syncErr := mmm.SyncDM(dc)
		set, getStsErr := mmm.setLister.StatefulSets(ns).Get(controller.DMWorkerMemberName(dcName))
		cmList := &corev1.ConfigMapList{}
		g.Expect(err).To(Succeed())
		listCmErr := controls.generic.FakeCli.List(context.TODO(), cmList)
		result := result{syncErr, oldSet, set, getStsErr, oldCm, cmList.Items, listCmErr}
		test.expectFn(g, &result)
	}

	tests := []*testcase{
		{
			name: "basic",
			prepare: func(tc *v1alpha1.DMCluster, _ *workerFakeIndexers) {
				tc.Spec.Worker.Config = &v1alpha1.WorkerConfig{
					LogLevel:     pointer.StringPtr("info"),
					KeepAliveTTL: pointer.Int64Ptr(25),
				}
			},
			expectFn: func(g *GomegaWithT, r *result) {
				g.Expect(r.sync).To(Succeed())
				g.Expect(r.listCm).To(Succeed())
				g.Expect(r.cms).To(HaveLen(2))
				g.Expect(r.getSet).To(Succeed())
				using := FindConfigMapVolume(&r.set.Spec.Template.Spec, func(name string) bool {
					return strings.HasPrefix(name, controller.DMWorkerMemberName("test"))
				})
				g.Expect(using).NotTo(BeEmpty())
				var usingCm *corev1.ConfigMap
				for _, cm := range r.cms {
					if cm.Name == using {
						usingCm = &cm
					}
				}
				g.Expect(usingCm).NotTo(BeNil(), "The configmap used by statefulset must be created")
				g.Expect(usingCm.Data["config-file"]).To(ContainSubstring("keepalive-ttl"),
					"The configmap used by statefulset should be the latest one")
			},
			workerInfos: []*dmapi.WorkersInfo{
				{Name: "worker1", Addr: "http://worker1:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: "worker2", Addr: "http://worker2:8262", Stage: v1alpha1.DMWorkerStateFree},
				{Name: "worker3", Addr: "http://worker3:8262", Stage: v1alpha1.DMWorkerStateFree},
			},
		},
	}

	for _, tt := range tests {
		testFn(tt, t)
	}
}

type workerFakeIndexers struct {
	svc cache.Indexer
	set cache.Indexer
	pod cache.Indexer
}

type workerFakeControls struct {
	svc     *controller.FakeServiceControl
	set     *controller.FakeStatefulSetControl
	generic *controller.FakeGenericControl
}

func newFakeWorkerMemberManager() (*workerMemberManager, *workerFakeControls, *workerFakeIndexers, *dmapi.FakeMasterControl) {
	// cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1().StatefulSets()
	// dcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().DMClusters()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	epsInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Endpoints()
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	setControl := controller.NewFakeStatefulSetControl(setInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, epsInformer)
	genericControl := controller.NewFakeGenericControl()
	masterControl := dmapi.NewFakeMasterControl(kubeCli)
	workerScaler := NewFakeWorkerScaler()
	autoFailover := true
	workerFailover := NewFakeWorkerFailover()
	pmm := &workerMemberManager{
		masterControl,
		setControl,
		svcControl,
		controller.NewTypedControl(genericControl),
		setInformer.Lister(),
		svcInformer.Lister(),
		podInformer.Lister(),
		workerScaler,
		autoFailover,
		workerFailover,
	}
	controls := &workerFakeControls{
		svc:     svcControl,
		set:     setControl,
		generic: genericControl,
	}
	indexers := &workerFakeIndexers{
		svc: svcInformer.Informer().GetIndexer(),
		set: setInformer.Informer().GetIndexer(),
		pod: podInformer.Informer().GetIndexer(),
	}

	return pmm, controls, indexers, masterControl
}

func newDMClusterForWorker() *v1alpha1.DMCluster {
	return &v1alpha1.DMCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DMCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.DMClusterSpec{
			Version: "v2.0.0-rc.2",
			Master: v1alpha1.MasterSpec{
				BaseImage:        "dm-test-image",
				Replicas:         1,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			Worker: &v1alpha1.WorkerSpec{
				BaseImage: "dm-test-image",
				Replicas:  3,
				Config: &v1alpha1.WorkerConfig{
					LogLevel:     pointer.StringPtr("debug"),
					KeepAliveTTL: pointer.Int64Ptr(15),
				},
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
		Status: v1alpha1.DMClusterStatus{
			Master: v1alpha1.MasterStatus{
				Synced: true,
				Members: map[string]v1alpha1.MasterMember{"test-dm-master-0": {
					Name:   "test-dm-master-0",
					Health: true,
				}},
				StatefulSet: &appsv1.StatefulSetStatus{
					ReadyReplicas: 1,
				},
			},
		},
	}
}

func TestGetNewWorkerHeadlessService(t *testing.T) {
	tests := []struct {
		name     string
		dc       v1alpha1.DMCluster
		expected corev1.Service
	}{
		{
			name: "basic",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-worker-peer",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-worker",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "DMCluster",
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
							Name:       "dm-worker",
							Port:       8262,
							TargetPort: intstr.FromInt(8262),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-worker",
					},
					PublishNotReadyAddresses: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewWorkerHeadlessServiceForDMCluster(&tt.dc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewWorkerSetForDMCluster(t *testing.T) {
	enable := true
	tests := []struct {
		name    string
		dc      v1alpha1.DMCluster
		wantErr bool
		testSts func(sts *appsv1.StatefulSet)
	}{
		{
			name: "dm-worker network is not host",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "dm-worker network is host",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
				},
			},
			testSts: testHostNetwork(t, true, v1.DNSClusterFirstWithHostNet),
		},
		{
			name: "dm-worker network is not host when dm-master is host",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "dm-worker should respect resources config",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Worker: &v1alpha1.WorkerSpec{
						ResourceRequirements: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:              resource.MustParse("1"),
								corev1.ResourceMemory:           resource.MustParse("2Gi"),
								corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
								corev1.ResourceStorage:          resource.MustParse("100Gi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:              resource.MustParse("1"),
								corev1.ResourceMemory:           resource.MustParse("2Gi"),
								corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
							},
						},
						StorageSize: "100Gi",
					},
					Master: v1alpha1.MasterSpec{},
				},
			},
			testSts: func(sts *appsv1.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.VolumeClaimTemplates[0].Spec.Resources).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				}))
				nameToContainer := MapContainers(&sts.Spec.Template.Spec)
				masterContainer := nameToContainer[v1alpha1.DMWorkerMemberType.String()]
				g.Expect(masterContainer.Resources).To(Equal(corev1.ResourceRequirements{
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
				}))
			},
		},
		{
			name: "set custom env",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Worker: &v1alpha1.WorkerSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							Env: []corev1.EnvVar{
								{
									Name:  "SOURCE1",
									Value: "mysql_replica1",
								},
								{
									Name:  "TZ",
									Value: "ignored",
								},
							},
						},
					},
					Master: v1alpha1.MasterSpec{},
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
					Name:  "CLUSTER_NAME",
					Value: "dc",
				},
				{
					Name:  "HEADLESS_SERVICE_NAME",
					Value: "dc-dm-worker-peer",
				},
				{
					Name:  "SET_NAME",
					Value: "dc-dm-worker",
				},
				{
					Name:  "TZ",
					Value: "UTC",
				},
				{
					Name:  "SOURCE1",
					Value: "mysql_replica1",
				},
			},
				v1alpha1.DMWorkerMemberType,
			),
		},
		{
			name: "dm version nightly, dm cluster tls is enabled",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-nightly",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{
						BaseImage: "pingcap/dm",
					},
					Version:    "nightly",
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
				},
			},
			testSts: func(sts *appsv1.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(hasClusterTLSVol(sts, "dm-worker-tls")).To(BeTrue())
				g.Expect(hasClusterVolMount(sts, v1alpha1.DMWorkerMemberType)).To(BeTrue())
			},
		},
		{
			name: "dmcluster worker with failureMember",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{
						BaseImage: "pingcap/dm",
						Replicas:  3,
					},
					Version: "nightly",
				},
				Status: v1alpha1.DMClusterStatus{
					Worker: v1alpha1.WorkerStatus{
						FailureMembers: map[string]v1alpha1.WorkerFailureMember{
							"test": {
								PodName: "test",
							},
						},
					},
				},
			},
			testSts: func(sts *appsv1.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(*sts.Spec.Replicas).To(Equal(int32(4)))
			},
		},
		{
			name: "dm-worker additional containers",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							AdditionalContainers: []corev1.Container{customSideCarContainers[0]},
						},
					},
				},
			},
			testSts: testAdditionalContainers(t, []corev1.Container{customSideCarContainers[0]}),
		},
		{
			name: "dm-worker additional volumes",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							AdditionalVolumes: []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
						},
					},
				},
			},
			testSts: testAdditionalVolumes(t, []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}),
		},
		// TODO add more tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, err := getNewWorkerSetForDMCluster(&tt.dc, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error %v, wantErr %v", err, tt.wantErr)
			}
			tt.testSts(sts)
		})
	}
}

func TestGetNewWorkerConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name     string
		dc       v1alpha1.DMCluster
		expected corev1.ConfigMap
	}{
		{
			name: "empty config",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Worker: &v1alpha1.WorkerSpec{
						Config: nil,
					},
				},
			},
			expected: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-worker",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-worker",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "DMCluster",
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
					"config-file":    "",
					"startup-script": "",
				},
			},
		},
		{
			name: "rolling update",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Worker: &v1alpha1.WorkerSpec{
						Config: &v1alpha1.WorkerConfig{
							LogLevel:     pointer.StringPtr("info"),
							KeepAliveTTL: pointer.Int64Ptr(25),
						},
					},
				},
			},
			expected: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-worker",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-worker",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "DMCluster",
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
					"config-file": `log-level = "info"
keepalive-ttl = 25
`,
					"startup-script": "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := getWorkerConfigMap(&tt.dc)
			g.Expect(err).To(Succeed())
			g.Expect(strings.HasPrefix(cm.Name, "foo-dm-worker")).To(BeTrue())
			tt.expected.Name = cm.Name
			// startup-script is better to be validated in e2e test
			cm.Data["startup-script"] = ""
			if diff := cmp.Diff(tt.expected, *cm); diff != "" {
				t.Errorf("unexpected ConfigMap (-want, +got): %s", diff)
			}
		})
	}
}
