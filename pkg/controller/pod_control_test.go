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

package controller

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestPodControlUpdateMetaInfoSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, recorder)
	pdClient := NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	pdClient.AddReaction(GetClusterActionType, func(action *Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(GetMembersActionType, func(action *Action) (interface{}, error) {
		membersInfo := &MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(GetStoresActionType, func(action *Action) (interface{}, error) {
		storesInfo := &StoresInfo{
			Stores: []*StoreInfo{
				{
					Store: &MetaStore{
						Store: &metapb.Store{
							Id:      333,
							Address: fmt.Sprintf("%s.web", TestPodName),
						},
					},
				},
			},
		}
		return storesInfo, nil
	})

	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestPodControlUpdateMetaInfoGetClusterFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, recorder)
	pdClient := NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	pdClient.AddReaction(GetClusterActionType, func(action *Action) (interface{}, error) {
		return nil, errors.New("failed to get cluster info from PD server")
	})
	pdClient.AddReaction(GetMembersActionType, func(action *Action) (interface{}, error) {
		membersInfo := &MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(GetStoresActionType, func(action *Action) (interface{}, error) {
		storesInfo := &StoresInfo{
			Stores: []*StoreInfo{
				{
					Store: &MetaStore{
						Store: &metapb.Store{
							Id:      333,
							Address: fmt.Sprintf("%s.web", TestPodName),
						},
					},
				},
			},
		}
		return storesInfo, nil
	})

	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(0))
}

func TestPodControlUpdateMetaInfoGetMemberFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, recorder)
	pdClient := NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	pdClient.AddReaction(GetClusterActionType, func(action *Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(GetMembersActionType, func(action *Action) (interface{}, error) {
		return nil, errors.New("failed to get member info from PD server")
	})
	pdClient.AddReaction(GetStoresActionType, func(action *Action) (interface{}, error) {
		storesInfo := &StoresInfo{
			Stores: []*StoreInfo{
				{
					Store: &MetaStore{
						Store: &metapb.Store{
							Id:      333,
							Address: fmt.Sprintf("%s.web", TestPodName),
						},
					},
				},
			},
		}
		return storesInfo, nil
	})

	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	pod.Labels[label.ComponentLabelKey] = label.PDLabelVal
	err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(0))
}

func TestPodControlUpdateMetaInfoGetStoreFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, recorder)
	pdClient := NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	pdClient.AddReaction(GetClusterActionType, func(action *Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(GetMembersActionType, func(action *Action) (interface{}, error) {
		membersInfo := &MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(GetStoresActionType, func(action *Action) (interface{}, error) {
		return nil, errors.New("failed to get store info from PD server")
	})

	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	pod.Labels[label.ComponentLabelKey] = label.TiKVLabelVal
	err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(0))
}

func TestPodControlUpdateMetaInfoUpdatePodFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, recorder)
	pdClient := NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	pdClient.AddReaction(GetClusterActionType, func(action *Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(GetMembersActionType, func(action *Action) (interface{}, error) {
		membersInfo := &MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(GetStoresActionType, func(action *Action) (interface{}, error) {
		storesInfo := &StoresInfo{
			Stores: []*StoreInfo{
				{
					Store: &MetaStore{
						Store: &metapb.Store{
							Id:      333,
							Address: fmt.Sprintf("%s.web", TestPodName),
						},
					},
				},
			},
		}
		return storesInfo, nil
	})

	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestPodControlUpdateMetaInfoConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, recorder)
	pdClient := NewFakePDClient()
	pdControl.SetPDClient(tc, pdClient)
	pdClient.AddReaction(GetClusterActionType, func(action *Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(GetMembersActionType, func(action *Action) (interface{}, error) {
		membersInfo := &MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(GetStoresActionType, func(action *Action) (interface{}, error) {
		storesInfo := &StoresInfo{
			Stores: []*StoreInfo{
				{
					Store: &MetaStore{
						Store: &metapb.Store{
							Id:      333,
							Address: fmt.Sprintf("%s.web", TestPodName),
						},
					},
				},
			},
		}
		return storesInfo, nil
	})

	conflict := false
	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, update.GetObject(), apierrors.NewConflict(action.GetResource().GroupResource(), pod.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func newFakeClientRecorderAndPDControl() (*fake.Clientset, *FakePDControl, *record.FakeRecorder) {
	fakeClient := &fake.Clientset{}
	pdControl := NewFakePDControl()
	recorder := record.NewFakeRecorder(10)
	return fakeClient, pdControl, recorder
}

func newPod(tc *v1alpha1.TidbCluster) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestPodName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
			Labels: map[string]string{
				label.ComponentLabelKey: TestComponentName,
				label.ManagedByLabelKey: TestManagedByName,
				label.InstanceLabelKey:  tc.GetName(),
			},
		},
		Spec: newPodSpec(v1alpha1.PDMemberType.String(), "pvc-1"),
	}
}

func newPodSpec(volumeName, pvcName string) corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "containerName",
				Image: "test",
				VolumeMounts: []corev1.VolumeMount{
					{Name: volumeName, MountPath: "/var/lib/test"},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			},
		},
	}
}
