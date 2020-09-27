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

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestMasterMemberManagerSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                           string
		prepare                        func(cluster *v1alpha1.DMCluster)
		errWhenCreateStatefulSet       bool
		errWhenCreateMasterService     bool
		errWhenCreateMasterPeerService bool
		errExpectFn                    func(*GomegaWithT, error)
		masterSvcCreated               bool
		masterPeerSvcCreated           bool
		setCreated                     bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		dc := newDMClusterForMaster()
		ns := dc.Namespace
		dcName := dc.Name
		oldSpec := dc.Spec
		if test.prepare != nil {
			test.prepare(dc)
		}

		mmm, fakeSetControl, fakeSvcControl, _, _, _, _ := newFakeMasterMemberManager()

		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateMasterService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateMasterPeerService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 1)
		}

		err := mmm.SyncDM(dc)
		test.errExpectFn(g, err)
		g.Expect(dc.Spec).To(Equal(oldSpec))

		svc1, err := mmm.svcLister.Services(ns).Get(controller.DMMasterMemberName(dcName))
		eps1, eperr := mmm.epsLister.Endpoints(ns).Get(controller.DMMasterMemberName(dcName))
		if test.masterSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc1).NotTo(Equal(nil))
			g.Expect(eperr).NotTo(HaveOccurred())
			g.Expect(eps1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
			expectErrIsNotFound(g, eperr)
		}

		svc2, err := mmm.svcLister.Services(ns).Get(controller.DMMasterPeerMemberName(dcName))
		eps2, eperr := mmm.epsLister.Endpoints(ns).Get(controller.DMMasterPeerMemberName(dcName))
		if test.masterPeerSvcCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc2).NotTo(Equal(nil))
			g.Expect(eperr).NotTo(HaveOccurred())
			g.Expect(eps2).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
			expectErrIsNotFound(g, eperr)
		}

		dc1, err := mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
		if test.setCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(dc1).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}
	}

	tests := []testcase{
		{
			name:                           "normal",
			prepare:                        nil,
			errWhenCreateStatefulSet:       false,
			errWhenCreateMasterService:     false,
			errWhenCreateMasterPeerService: false,
			errExpectFn:                    errExpectRequeue,
			masterSvcCreated:               true,
			masterPeerSvcCreated:           true,
			setCreated:                     true,
		},
		{
			name:                           "error when create statefulset",
			prepare:                        nil,
			errWhenCreateStatefulSet:       true,
			errWhenCreateMasterService:     false,
			errWhenCreateMasterPeerService: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server failed")).To(BeTrue())
			},
			masterSvcCreated:     true,
			masterPeerSvcCreated: true,
			setCreated:           false,
		},
		{
			name:                           "error when create dm-master service",
			prepare:                        nil,
			errWhenCreateStatefulSet:       false,
			errWhenCreateMasterService:     true,
			errWhenCreateMasterPeerService: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server failed")).To(BeTrue())
			},
			masterSvcCreated:     false,
			masterPeerSvcCreated: false,
			setCreated:           false,
		},
		{
			name:                           "error when create dm-master peer service",
			prepare:                        nil,
			errWhenCreateStatefulSet:       false,
			errWhenCreateMasterService:     false,
			errWhenCreateMasterPeerService: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server failed")).To(BeTrue())
			},
			masterSvcCreated:     true,
			masterPeerSvcCreated: false,
			setCreated:           false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestMasterMemberManagerSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                       string
		modify                     func(cluster *v1alpha1.DMCluster)
		leaderInfo                 dmapi.MembersLeader
		masterInfos                []*dmapi.MastersInfo
		errWhenUpdateStatefulSet   bool
		errWhenUpdateMasterService bool
		errWhenGetLeader           bool
		errWhenGetMasterInfos      bool
		statusChange               func(*apps.StatefulSet)
		err                        bool
		expectMasterServiceFn      func(*GomegaWithT, *corev1.Service, error)
		expectMasterPeerServiceFn  func(*GomegaWithT, *corev1.Service, error)
		expectStatefulSetFn        func(*GomegaWithT, *apps.StatefulSet, error)
		expectDMClusterFn          func(*GomegaWithT, *v1alpha1.DMCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		dc := newDMClusterForMaster()
		ns := dc.Namespace
		dcName := dc.Name

		mmm, fakeSetControl, fakeSvcControl, fakeMasterControl, _, _, _ := newFakeMasterMemberManager()
		masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)
		if test.errWhenGetMasterInfos {
			masterClient.AddReaction(dmapi.GetMastersActionType, func(action *dmapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get master infos of dm-master cluster")
			})
		} else {
			masterClient.AddReaction(dmapi.GetMastersActionType, func(action *dmapi.Action) (interface{}, error) {
				return test.masterInfos, nil
			})
		}
		if test.errWhenGetLeader {
			masterClient.AddReaction(dmapi.GetLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("failed to get leader info of dm-master cluster")
			})
		} else {
			masterClient.AddReaction(dmapi.GetLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
				return test.leaderInfo, nil
			})
		}

		if test.statusChange == nil {
			fakeSetControl.SetStatusChange(func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "dm-master-1"
				set.Status.UpdateRevision = "dm-master-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			})
		} else {
			fakeSetControl.SetStatusChange(test.statusChange)
		}

		err := mmm.SyncDM(dc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.epsLister.Endpoints(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())

		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterPeerMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.epsLister.Endpoints(ns).Get(controller.DMMasterPeerMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())

		_, err = mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())

		dc1 := dc.DeepCopy()
		test.modify(dc1)

		if test.errWhenUpdateMasterService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenUpdateStatefulSet {
			fakeSetControl.SetUpdateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = mmm.SyncDM(dc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectMasterServiceFn != nil {
			svc, err := mmm.svcLister.Services(ns).Get(controller.DMMasterMemberName(dcName))
			test.expectMasterServiceFn(g, svc, err)
		}
		if test.expectMasterPeerServiceFn != nil {
			svc, err := mmm.svcLister.Services(ns).Get(controller.DMMasterPeerMemberName(dcName))
			test.expectMasterPeerServiceFn(g, svc, err)
		}
		if test.expectStatefulSetFn != nil {
			set, err := mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectDMClusterFn != nil {
			test.expectDMClusterFn(g, dc1)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(dc *v1alpha1.DMCluster) {
				dc.Spec.Master.Replicas = 5
				masterNodePort := 30160
				dc.Spec.Master.Service = &v1alpha1.MasterServiceSpec{MasterNodePort: &masterNodePort}
				dc.Spec.Master.Service.Type = corev1.ServiceTypeNodePort
			},
			leaderInfo: dmapi.MembersLeader{
				Name: "master1",
				Addr: "http://master1:2379",
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "master1", MemberID: "1", ClientURLs: []string{"http://master1:2379"}, Alive: true},
				{Name: "master2", MemberID: "2", ClientURLs: []string{"http://master2:2379"}, Alive: true},
				{Name: "master3", MemberID: "3", ClientURLs: []string{"http://master3:2379"}, Alive: false},
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdateMasterService: false,
			errWhenGetLeader:           false,
			errWhenGetMasterInfos:      false,
			err:                        false,
			expectMasterServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectMasterPeerServiceFn: func(g *GomegaWithT, svc *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.ScalePhase))
				g.Expect(dc.Status.Master.StatefulSet.ObservedGeneration).To(Equal(int64(1)))
				g.Expect(len(dc.Status.Master.Members)).To(Equal(3))
				g.Expect(dc.Status.Master.Members["master1"].Health).To(Equal(true))
				g.Expect(dc.Status.Master.Members["master2"].Health).To(Equal(true))
				g.Expect(dc.Status.Master.Members["master3"].Health).To(Equal(false))
			},
		},
		{
			name: "error when update dm-master service",
			modify: func(dc *v1alpha1.DMCluster) {
				masterNodePort := 30160
				dc.Spec.Master.Service = &v1alpha1.MasterServiceSpec{MasterNodePort: &masterNodePort}
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "master1", MemberID: "1", ClientURLs: []string{"http://master1:2379"}, Alive: true},
				{Name: "master2", MemberID: "2", ClientURLs: []string{"http://master2:2379"}, Alive: true},
				{Name: "master3", MemberID: "3", ClientURLs: []string{"http://master3:2379"}, Alive: false},
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdateMasterService: true,
			errWhenGetLeader:           false,
			errWhenGetMasterInfos:      false,
			err:                        true,
			expectMasterServiceFn:      nil,
			expectMasterPeerServiceFn:  nil,
			expectStatefulSetFn:        nil,
		},
		{
			name: "error when update statefulset",
			modify: func(dc *v1alpha1.DMCluster) {
				dc.Spec.Master.Replicas = 5
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "master1", MemberID: "1", ClientURLs: []string{"http://master1:2379"}, Alive: true},
				{Name: "master2", MemberID: "2", ClientURLs: []string{"http://master2:2379"}, Alive: true},
				{Name: "master3", MemberID: "3", ClientURLs: []string{"http://master3:2379"}, Alive: false},
			},
			errWhenUpdateStatefulSet:   true,
			errWhenUpdateMasterService: false,
			errWhenGetLeader:           false,
			errWhenGetMasterInfos:      false,
			err:                        true,
			expectMasterServiceFn:      nil,
			expectMasterPeerServiceFn:  nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "error when get dm-master leader",
			modify: func(dc *v1alpha1.DMCluster) {
				dc.Spec.Master.Replicas = 5
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdateMasterService: false,
			errWhenGetLeader:           true,
			errWhenGetMasterInfos:      false,
			err:                        false,
			expectMasterServiceFn:      nil,
			expectMasterPeerServiceFn:  nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.Synced).To(BeFalse())
				g.Expect(dc.Status.Master.Members).To(BeNil())
			},
		},
		{
			name: "error when sync dm-master infos",
			modify: func(dc *v1alpha1.DMCluster) {
				dc.Spec.Master.Replicas = 5
			},
			errWhenUpdateStatefulSet:   false,
			errWhenUpdateMasterService: false,
			errWhenGetLeader:           false,
			errWhenGetMasterInfos:      true,
			err:                        false,
			expectMasterServiceFn:      nil,
			expectMasterPeerServiceFn:  nil,
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.Synced).To(BeFalse())
				g.Expect(dc.Status.Master.Members).To(BeNil())
			},
		},
	}

	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestMasterMemberManagerMasterStatefulSetIsUpgrading(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name            string
		setUpdate       func(*apps.StatefulSet)
		hasPod          bool
		updatePod       func(*corev1.Pod)
		errExpectFn     func(*GomegaWithT, error)
		expectUpgrading bool
	}
	testFn := func(test *testcase, t *testing.T) {
		mmm, _, _, _, podIndexer, _, _ := newFakeMasterMemberManager()
		dc := newDMClusterForMaster()
		dc.Status.Master.StatefulSet = &apps.StatefulSetStatus{
			UpdateRevision: "v3",
		}

		set := &apps.StatefulSet{
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
					Name:        ordinalPodName(v1alpha1.DMMasterMemberType, dc.GetName(), 0),
					Namespace:   metav1.NamespaceDefault,
					Annotations: map[string]string{},
					Labels:      label.NewDM().Instance(dc.GetInstanceName()).DMMaster().Labels(),
				},
			}
			if test.updatePod != nil {
				test.updatePod(pod)
			}
			podIndexer.Add(pod)
		}
		b, err := mmm.masterStatefulSetIsUpgrading(set, dc)
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
			setUpdate: func(set *apps.StatefulSet) {
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
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v2"
			},
			errExpectFn:     nil,
			expectUpgrading: true,
		},
		{
			name:      "pod have revision hash, equal statefulset's",
			setUpdate: nil,
			hasPod:    true,
			updatePod: func(pod *corev1.Pod) {
				pod.Labels[apps.ControllerRevisionHashLabelKey] = "v3"
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

func TestMasterMemberManagerUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.DMCluster)
		leaderInfo          dmapi.MembersLeader
		masterInfos         []*dmapi.MastersInfo
		err                 bool
		statusChange        func(*apps.StatefulSet)
		expectStatefulSetFn func(*GomegaWithT, *apps.StatefulSet, error)
		expectDMClusterFn   func(*GomegaWithT, *v1alpha1.DMCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		dc := newDMClusterForMaster()
		ns := dc.Namespace
		dcName := dc.Name

		mmm, fakeSetControl, _, fakeMasterControl, _, _, _ := newFakeMasterMemberManager()
		masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)
		masterClient.AddReaction(dmapi.GetMastersActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.masterInfos, nil
		})
		masterClient.AddReaction(dmapi.GetLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.leaderInfo, nil
		})

		fakeSetControl.SetStatusChange(test.statusChange)

		err := mmm.SyncDM(dc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterPeerMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())

		dc1 := dc.DeepCopy()
		test.modify(dc1)

		err = mmm.SyncDM(dc1)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
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
				cluster.Spec.Master.BaseImage = "dm-test-image-2"
			},
			leaderInfo: dmapi.MembersLeader{
				Name: "master1",
				Addr: "http://master1:8261",
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "master1", MemberID: "1", ClientURLs: []string{"http://master1:8261"}, Alive: true},
				{Name: "master2", MemberID: "2", ClientURLs: []string{"http://master2:8261"}, Alive: true},
				{Name: "master3", MemberID: "3", ClientURLs: []string{"http://master3:8261"}, Alive: false},
			},
			err: false,
			statusChange: func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "dm-master-1"
				set.Status.UpdateRevision = "dm-master-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("dm-test-image-2:v2.0.0-rc.2"))
			},
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
				g.Expect(len(dc.Status.Master.Members)).To(Equal(3))
				g.Expect(dc.Status.Master.Members["master1"].Health).To(Equal(true))
				g.Expect(dc.Status.Master.Members["master2"].Health).To(Equal(true))
				g.Expect(dc.Status.Master.Members["master3"].Health).To(Equal(false))
			},
		},
	}
	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestMasterMemberManagerSyncMasterSts(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		modify              func(cluster *v1alpha1.DMCluster)
		leaderInfo          dmapi.MembersLeader
		masterInfos         []*dmapi.MastersInfo
		err                 bool
		statusChange        func(*apps.StatefulSet)
		expectStatefulSetFn func(*GomegaWithT, *apps.StatefulSet, error)
		expectDMClusterFn   func(*GomegaWithT, *v1alpha1.DMCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		dc := newDMClusterForMaster()
		ns := dc.Namespace
		dcName := dc.Name

		mmm, fakeSetControl, _, fakeMasterControl, _, _, _ := newFakeMasterMemberManager()
		masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)
		masterClient.AddReaction(dmapi.GetMastersActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.masterInfos, nil
		})
		masterClient.AddReaction(dmapi.GetLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.leaderInfo, nil
		})

		fakeSetControl.SetStatusChange(test.statusChange)

		err := mmm.SyncDM(dc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())

		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterPeerMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())

		test.modify(dc)
		masterClient.AddReaction(dmapi.GetLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
			return nil, fmt.Errorf("cannot get leader")
		})
		err = mmm.syncMasterStatefulSetForDMCluster(dc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.expectStatefulSetFn != nil {
			set, err := mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
			test.expectStatefulSetFn(g, set, err)
		}
		if test.expectDMClusterFn != nil {
			test.expectDMClusterFn(g, dc)
		}
	}
	tests := []testcase{
		{
			name: "force upgrade",
			modify: func(cluster *v1alpha1.DMCluster) {
				cluster.Spec.Master.BaseImage = "dm-test-image-2"
				cluster.Spec.Master.Replicas = 1
				cluster.ObjectMeta.Annotations = make(map[string]string)
				cluster.ObjectMeta.Annotations["tidb.pingcap.com/force-upgrade"] = "true"
			},
			leaderInfo: dmapi.MembersLeader{
				Name: "master1",
				Addr: "http://master1:8261",
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "master1", MemberID: "1", ClientURLs: []string{"http://master1:8261"}, Alive: true},
				{Name: "master2", MemberID: "2", ClientURLs: []string{"http://master2:8261"}, Alive: true},
				{Name: "master3", MemberID: "3", ClientURLs: []string{"http://master3:8261"}, Alive: false},
			},
			err: true,
			statusChange: func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "dm-master-1"
				set.Status.UpdateRevision = "dm-master-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("dm-test-image-2:v2.0.0-rc.2"))
				// scale in one pd from 3 -> 2
				g.Expect(*set.Spec.Replicas).To(Equal(int32(2)))
				g.Expect(*set.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(0)))
			},
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.UpgradePhase))
			},
		},
		{
			name: "non force upgrade",
			modify: func(cluster *v1alpha1.DMCluster) {
				cluster.Spec.Master.BaseImage = "dm-test-image-2"
				cluster.Spec.Master.Replicas = 1
			},
			leaderInfo: dmapi.MembersLeader{
				Name: "master1",
				Addr: "http://master1:8261",
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "master1", MemberID: "1", ClientURLs: []string{"http://master1:8261"}, Alive: true},
				{Name: "master2", MemberID: "2", ClientURLs: []string{"http://master2:8261"}, Alive: true},
				{Name: "master3", MemberID: "3", ClientURLs: []string{"http://master3:8261"}, Alive: false},
			},
			err: true,
			statusChange: func(set *apps.StatefulSet) {
				set.Status.Replicas = *set.Spec.Replicas
				set.Status.CurrentRevision = "dm-master-1"
				set.Status.UpdateRevision = "dm-master-1"
				observedGeneration := int64(1)
				set.Status.ObservedGeneration = observedGeneration
			},
			expectStatefulSetFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set.Spec.Template.Spec.Containers[0].Image).To(Equal("dm-test-image:v2.0.0-rc.2"))
				g.Expect(*set.Spec.Replicas).To(Equal(int32(3)))
				g.Expect(*set.Spec.UpdateStrategy.RollingUpdate.Partition).To(Equal(int32(3)))
			},
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.Phase).To(Equal(v1alpha1.ScalePhase))
			},
		},
	}
	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func newFakeMasterMemberManager() (*masterMemberManager, *controller.FakeStatefulSetControl, *controller.FakeServiceControl, *dmapi.FakeMasterControl, cache.Indexer, cache.Indexer, *controller.FakePodControl) {
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	epsInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Endpoints()
	pvcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().PersistentVolumeClaims()
	setControl := controller.NewFakeStatefulSetControl(setInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, epsInformer)
	podControl := controller.NewFakePodControl(podInformer)
	masterControl := dmapi.NewFakeMasterControl(kubeCli)
	masterScaler := NewFakeMasterScaler()
	autoFailover := true
	masterFailover := NewFakeMasterFailover()
	masterUpgrader := NewFakeMasterUpgrader()
	genericControll := controller.NewFakeGenericControl()

	return &masterMemberManager{
		masterControl,
		setControl,
		svcControl,
		controller.NewTypedControl(genericControll),
		setInformer.Lister(),
		svcInformer.Lister(),
		podInformer.Lister(),
		epsInformer.Lister(),
		pvcInformer.Lister(),
		masterScaler,
		masterUpgrader,
		autoFailover,
		masterFailover,
	}, setControl, svcControl, masterControl, podInformer.Informer().GetIndexer(), pvcInformer.Informer().GetIndexer(), podControl
}

func newDMClusterForMaster() *v1alpha1.DMCluster {
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
			Version:   "v2.0.0-rc.2",
			Discovery: v1alpha1.DMDiscoverySpec{Address: "http://basic-discovery.demo:10261"},
			Master: v1alpha1.MasterSpec{
				BaseImage:        "dm-test-image",
				StorageSize:      "100Gi",
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			Worker: &v1alpha1.WorkerSpec{
				BaseImage:        "dm-test-image",
				StorageSize:      "100Gi",
				Replicas:         3,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
		},
	}
}

func TestGetNewMasterHeadlessServiceForDMCluster(t *testing.T) {
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
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master-peer",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
						"app.kubernetes.io/used-by":    "peer",
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
							Name:       "dm-master-peer",
							Port:       8291,
							TargetPort: intstr.FromInt(8291),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
					},
					PublishNotReadyAddresses: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := getNewMasterHeadlessServiceForDMCluster(&tt.dc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewMasterSetForDMCluster(t *testing.T) {
	enable := true
	tests := []struct {
		name    string
		dc      v1alpha1.DMCluster
		wantErr bool
		testSts func(sts *apps.StatefulSet)
	}{
		{
			name: "dm-master network is not host",
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
			name: "dm-master network is host",
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
			testSts: testHostNetwork(t, true, v1.DNSClusterFirstWithHostNet),
		},
		{
			name: "dm-master network is not host when dm-worker is host",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Worker: &v1alpha1.WorkerSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							HostNetwork: &enable,
						},
					},
					Master: v1alpha1.MasterSpec{},
				},
			},
			testSts: testHostNetwork(t, false, ""),
		},
		{
			name: "dm-master should respect resources config",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
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
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(sts.Spec.VolumeClaimTemplates[0].Spec.Resources).To(Equal(corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				}))
				nameToContainer := MapContainers(&sts.Spec.Template.Spec)
				masterContainer := nameToContainer[v1alpha1.DMMasterMemberType.String()]
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
					Master: v1alpha1.MasterSpec{
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
					Worker: &v1alpha1.WorkerSpec{},
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
					Value: "dc-dm-master-peer",
				},
				{
					Name:  "SERVICE_NAME",
					Value: "dc-dm-master",
				},
				{
					Name:  "SET_NAME",
					Value: "dc-dm-master",
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
				v1alpha1.DMMasterMemberType,
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
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
					},
					Worker:     &v1alpha1.WorkerSpec{},
					Version:    "nightly",
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(hasClusterTLSVol(sts, "dm-master-tls")).To(BeTrue())
				g.Expect(hasClusterVolMount(sts, v1alpha1.DMMasterMemberType)).To(BeTrue())
			},
		},
		{
			name: "dmcluster with failureMember nonDeleted",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
						Replicas:  3,
					},
					Worker:  &v1alpha1.WorkerSpec{},
					Version: "nightly",
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						FailureMembers: map[string]v1alpha1.MasterFailureMember{
							"test": {
								MemberDeleted: false,
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(*sts.Spec.Replicas).To(Equal(int32(3)))
			},
		},
		{
			name: "dmcluster with failureMember Deleted",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
						Replicas:  3,
					},
					Worker:  &v1alpha1.WorkerSpec{},
					Version: "nightly",
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						FailureMembers: map[string]v1alpha1.MasterFailureMember{
							"test": {
								MemberDeleted: true,
							},
						},
					},
				},
			},
			testSts: func(sts *apps.StatefulSet) {
				g := NewGomegaWithT(t)
				g.Expect(*sts.Spec.Replicas).To(Equal(int32(4)))
			},
		},
		{
			name: "dm-master additional containers",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							AdditionalContainers: []corev1.Container{customSideCarContainers[0]},
						},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			testSts: testAdditionalContainers(t, []corev1.Container{customSideCarContainers[0]}),
		},
		{
			name: "dm-master additional volumes",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dc",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						ComponentSpec: v1alpha1.ComponentSpec{
							AdditionalVolumes: []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
						},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			testSts: testAdditionalVolumes(t, []corev1.Volume{{Name: "test", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}),
		},
		// TODO add more tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts, err := getNewMasterSetForDMCluster(&tt.dc, &corev1.ConfigMap{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error %v, wantErr %v", err, tt.wantErr)
			}
			tt.testSts(sts)
		})
	}
}

func TestGetMasterConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	testCases := []struct {
		name     string
		dc       v1alpha1.DMCluster
		expected *corev1.ConfigMap
	}{
		{
			name: "dm-master config is nil",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
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
					"startup-script": "",
					"config-file":    "",
				},
			},
		},
		{
			name: "basic",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Config: &v1alpha1.MasterConfig{
							LogLevel:      pointer.StringPtr("debug"),
							RPCTimeoutStr: pointer.StringPtr("40s"),
							RPCRateLimit:  pointer.Float64Ptr(15),
						},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
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
					"startup-script": "",
					"config-file": `log-level = "debug"
rpc-timeout = "40s"
rpc-rate-limit = 15.0
`,
				},
			},
		},
		{
			name: "dm version v2.0.0-rc.2, dm cluster tls is enabled",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v2",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						BaseImage: "pingcap/dm",
					},
					Worker:     &v1alpha1.WorkerSpec{},
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
					Version:    "v2.0.0-rc.2",
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-v2-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "tls-v2",
						"app.kubernetes.io/component":  "dm-master",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "pingcap.com/v1alpha1",
							Kind:       "DMCluster",
							Name:       "tls-v2",
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
					"config-file": `ssl-ca = "/var/lib/dm-master-tls/ca.crt"
ssl-cert = "/var/lib/dm-master-tls/tls.crt"
ssl-key = "/var/lib/dm-master-tls/tls.key"
`,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := getMasterConfigMap(&tt.dc)
			g.Expect(err).To(Succeed())
			// startup-script is better to be tested in e2e
			tt.expected.Data["startup-script"] = cm.Data["startup-script"]
			g.Expect(AddConfigMapDigestSuffix(tt.expected)).To(Succeed())
			if diff := cmp.Diff(*tt.expected, *cm); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetNewMasterServiceForDMCluster(t *testing.T) {
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
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
						"app.kubernetes.io/used-by":    "end-user",
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
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "dm-master",
							Port:       8261,
							TargetPort: intstr.FromInt(8261),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
					},
				},
			},
		},
		{
			name: "basic and specify ClusterIP type,clusterIP",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Service: &v1alpha1.MasterServiceSpec{ServiceSpec: v1alpha1.ServiceSpec{ClusterIP: pointer.StringPtr("172.20.10.1")}},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
						"app.kubernetes.io/used-by":    "end-user",
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
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "dm-master",
							Port:       8261,
							TargetPort: intstr.FromInt(8261),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
					},
				},
			},
		},
		{
			name: "basic and specify LoadBalancerIP type, LoadBalancerType",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Service: &v1alpha1.MasterServiceSpec{
							ServiceSpec: v1alpha1.ServiceSpec{
								Type:           corev1.ServiceTypeLoadBalancer,
								LoadBalancerIP: pointer.StringPtr("172.20.10.1"),
							}},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
						"app.kubernetes.io/used-by":    "end-user",
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
					LoadBalancerIP: "172.20.10.1",
					Type:           corev1.ServiceTypeLoadBalancer,
					Ports: []corev1.ServicePort{
						{
							Name:       "dm-master",
							Port:       8261,
							TargetPort: intstr.FromInt(8261),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
					},
				},
			},
		},
		{
			name: "basic and specify dm-master service NodePort",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Service: &v1alpha1.MasterServiceSpec{
							ServiceSpec: v1alpha1.ServiceSpec{
								Type:      corev1.ServiceTypeNodePort,
								ClusterIP: pointer.StringPtr("172.20.10.1"),
							},
							MasterNodePort: intPtr(30020),
						},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
						"app.kubernetes.io/used-by":    "end-user",
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
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Name:       "dm-master",
							Port:       8261,
							TargetPort: intstr.FromInt(8261),
							NodePort:   30020,
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
					},
				},
			},
		},
		{
			name: "basic and specify dm-master service portname",
			dc: v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Service: &v1alpha1.MasterServiceSpec{
							ServiceSpec: v1alpha1.ServiceSpec{
								Type:      corev1.ServiceTypeClusterIP,
								ClusterIP: pointer.StringPtr("172.20.10.1"),
								PortName:  pointer.StringPtr("http-dm-master"),
							},
						},
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
			},
			expected: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dm-master",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
						"app.kubernetes.io/used-by":    "end-user",
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
					ClusterIP: "172.20.10.1",
					Type:      corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:       "http-dm-master",
							Port:       8261,
							TargetPort: intstr.FromInt(8261),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: map[string]string{
						"app.kubernetes.io/name":       "dm-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "dm-master",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmm, _, _, _, _, _, _ := newFakeMasterMemberManager()
			svc := mmm.getNewMasterServiceForDMCluster(&tt.dc)
			if diff := cmp.Diff(tt.expected, *svc); diff != "" {
				t.Errorf("unexpected Service (-want, +got): %s", diff)
			}
		})
	}
}

func TestMasterMemberManagerSyncMasterStsWhenMasterNotJoinCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name              string
		modify            func(cluster *v1alpha1.DMCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer)
		leaderInfo        dmapi.MembersLeader
		masterInfos       []*dmapi.MastersInfo
		dcStatusChange    func(cluster *v1alpha1.DMCluster)
		err               bool
		expectDMClusterFn func(*GomegaWithT, *v1alpha1.DMCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		dc := newDMClusterForMaster()
		ns := dc.Namespace
		dcName := dc.Name

		mmm, _, _, fakeMasterControl, podIndexer, pvcIndexer, _ := newFakeMasterMemberManager()
		masterClient := controller.NewFakeMasterClient(fakeMasterControl, dc)

		masterClient.AddReaction(dmapi.GetMastersActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.masterInfos, nil
		})
		masterClient.AddReaction(dmapi.GetLeaderActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.leaderInfo, nil
		})

		err := mmm.SyncDM(dc)
		g.Expect(controller.IsRequeueError(err)).To(BeTrue())
		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.svcLister.Services(ns).Get(controller.DMMasterPeerMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.setLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
		g.Expect(err).NotTo(HaveOccurred())
		if test.dcStatusChange != nil {
			test.dcStatusChange(dc)
		}
		test.modify(dc, podIndexer, pvcIndexer)
		err = mmm.syncMasterStatefulSetForDMCluster(dc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.expectDMClusterFn != nil {
			test.expectDMClusterFn(g, dc)
		}
	}
	tests := []testcase{
		{
			name: "add dm-master unjoin cluster member info",
			modify: func(cluster *v1alpha1.DMCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				for ordinal := 0; ordinal < 3; ordinal++ {
					pod := &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPodName(v1alpha1.DMMasterMemberType, cluster.GetName(), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.NewDM().Instance(cluster.GetInstanceName()).DMMaster().Labels(),
						},
					}
					podIndexer.Add(pod)
				}
				for ordinal := 0; ordinal < 3; ordinal++ {
					pvc := &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPVCName(v1alpha1.DMMasterMemberType, controller.DMMasterMemberName(cluster.GetName()), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.NewDM().Instance(cluster.GetInstanceName()).DMMaster().Labels(),
						},
					}
					pvcIndexer.Add(pvc)
				}

			},
			leaderInfo: dmapi.MembersLeader{
				Name: "test-dm-master-0",
				Addr: "http://test-dm-master-0:8261",
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "test-dm-master-0", MemberID: "1", ClientURLs: []string{"http://test-dm-master-0:8261"}, Alive: false},
				{Name: "test-dm-master-1", MemberID: "2", ClientURLs: []string{"http://test-dm-master-1:8261"}, Alive: false},
			},
			err: false,
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.UnjoinedMembers["test-dm-master-2"]).NotTo(BeNil())
			},
		},
		{
			name: "clear unjoin cluster member info when the member join the cluster",
			dcStatusChange: func(cluster *v1alpha1.DMCluster) {
				cluster.Status.Master.UnjoinedMembers = map[string]v1alpha1.UnjoinedMember{
					"test-dm-master-0": {
						PodName:   "test-dm-master-0",
						CreatedAt: metav1.Now(),
					},
				}
			},
			modify: func(cluster *v1alpha1.DMCluster, podIndexer cache.Indexer, pvcIndexer cache.Indexer) {
				for ordinal := 0; ordinal < 3; ordinal++ {
					pod := &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPodName(v1alpha1.DMMasterMemberType, cluster.GetName(), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.NewDM().Instance(cluster.GetInstanceName()).DMMaster().Labels(),
						},
					}
					podIndexer.Add(pod)
				}
				for ordinal := 0; ordinal < 3; ordinal++ {
					pvc := &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:        ordinalPVCName(v1alpha1.DMMasterMemberType, controller.DMMasterMemberName(cluster.GetName()), int32(ordinal)),
							Namespace:   metav1.NamespaceDefault,
							Annotations: map[string]string{},
							Labels:      label.NewDM().Instance(cluster.GetInstanceName()).DMMaster().Labels(),
						},
					}
					pvcIndexer.Add(pvc)
				}
			},
			leaderInfo: dmapi.MembersLeader{
				Name: "test-dm-master-0",
				Addr: "http://test-dm-master-0:8261",
			},
			masterInfos: []*dmapi.MastersInfo{
				{Name: "test-dm-master-0", MemberID: "1", ClientURLs: []string{"http://test-dm-master-0:8261"}, Alive: false},
				{Name: "test-dm-master-1", MemberID: "2", ClientURLs: []string{"http://test-dm-master-1:8261"}, Alive: false},
				{Name: "test-dm-master-2", MemberID: "3", ClientURLs: []string{"http://test-dm-master-2:8261"}, Alive: false},
			},
			err: false,
			expectDMClusterFn: func(g *GomegaWithT, dc *v1alpha1.DMCluster) {
				g.Expect(dc.Status.Master.UnjoinedMembers).To(BeEmpty())
			},
		},
	}
	for i := range tests {
		t.Logf("begin: %s", tests[i].name)
		testFn(&tests[i], t)
		t.Logf("end: %s", tests[i].name)
	}
}

func TestMasterShouldRecover(t *testing.T) {
	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-dm-master-0",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failover-dm-master-1",
				Namespace: v1.NamespaceDefault,
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	}
	podsWithFailover := append(pods, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failover-dm-master-2",
			Namespace: v1.NamespaceDefault,
		},
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	})
	tests := []struct {
		name string
		dc   *v1alpha1.DMCluster
		pods []*v1.Pod
		want bool
	}{
		{
			name: "should not recover if no failure members",
			dc: &v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Status: v1alpha1.DMClusterStatus{},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should not recover if a member is not healthy",
			dc: &v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						Members: map[string]v1alpha1.MasterMember{
							"failover-dm-master-0": {
								Name:   "failover-dm-master-0",
								Health: false,
							},
							"failover-dm-master-1": {
								Name:   "failover-dm-master-1",
								Health: true,
							},
						},
						FailureMembers: map[string]v1alpha1.MasterFailureMember{
							"failover-dm-master-0": {
								PodName: "failover-dm-master-0",
							},
						},
					},
				},
			},
			pods: pods,
			want: false,
		},
		{
			name: "should recover if all members are ready and healthy",
			dc: &v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						Members: map[string]v1alpha1.MasterMember{
							"failover-dm-master-0": {
								Name:   "failover-dm-master-0",
								Health: true,
							},
							"failover-dm-master-1": {
								Name:   "failover-dm-master-1",
								Health: true,
							},
						},
						FailureMembers: map[string]v1alpha1.MasterFailureMember{
							"failover-dm-master-0": {
								PodName: "failover-dm-master-0",
							},
						},
					},
				},
			},
			pods: pods,
			want: true,
		},
		{
			name: "should recover if all members are ready and healthy (ignore auto-created failover pods)",
			dc: &v1alpha1.DMCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failover",
					Namespace: v1.NamespaceDefault,
				},
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Replicas: 2,
					},
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						Members: map[string]v1alpha1.MasterMember{
							"failover-dm-master-0": {
								Name:   "failover-dm-master-0",
								Health: true,
							},
							"failover-dm-master-1": {
								Name:   "failover-dm-master-1",
								Health: true,
							},
							"failover-dm-master-2": {
								Name:   "failover-dm-master-1",
								Health: false,
							},
						},
						FailureMembers: map[string]v1alpha1.MasterFailureMember{
							"failover-dm-master-0": {
								PodName: "failover-dm-master-0",
							},
						},
					},
				},
			},
			pods: podsWithFailover,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := kubefake.NewSimpleClientset()
			for _, pod := range tt.pods {
				client.CoreV1().Pods(pod.Namespace).Create(pod)
			}
			kubeInformerFactory := kubeinformers.NewSharedInformerFactory(client, 0)
			podLister := kubeInformerFactory.Core().V1().Pods().Lister()
			kubeInformerFactory.Start(ctx.Done())
			kubeInformerFactory.WaitForCacheSync(ctx.Done())
			masterMemberManager := &masterMemberManager{podLister: podLister}
			got := masterMemberManager.shouldRecover(tt.dc)
			if got != tt.want {
				t.Fatalf("wants %v, got %v", tt.want, got)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func hasClusterTLSVol(sts *apps.StatefulSet, volName string) bool {
	for _, vol := range sts.Spec.Template.Spec.Volumes {
		if vol.Name == volName {
			return true
		}
	}
	return false
}

func hasClusterVolMount(sts *apps.StatefulSet, memberType v1alpha1.MemberType) bool {
	var vmName string
	switch memberType {
	case v1alpha1.DMMasterMemberType:
		vmName = "dm-master-tls"
	case v1alpha1.DMWorkerMemberType:
		vmName = "dm-worker-tls"
	default:
		return false
	}
	for _, container := range sts.Spec.Template.Spec.Containers {
		if container.Name == memberType.String() {
			for _, vm := range container.VolumeMounts {
				if vm.Name == vmName {
					return true
				}
			}
		}
	}
	return false
}
