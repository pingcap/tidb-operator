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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestPodControlUpdateMetaInfoSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, podLister, _, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	pdClient := NewFakePDClient(pdControl, tc)
	pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		membersInfo := &pdapi.MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		storesInfo := &pdapi.StoresInfo{
			Stores: []*pdapi.StoreInfo{
				{
					Store: &pdapi.MetaStore{
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
	_, err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(Succeed())
}

func TestPodControlUpdateMetaInfoGetClusterFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, podLister, _, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	pdClient := NewFakePDClient(pdControl, tc)
	pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
		return nil, errors.New("failed to get cluster info from PD server")
	})
	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		membersInfo := &pdapi.MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		storesInfo := &pdapi.StoresInfo{
			Stores: []*pdapi.StoreInfo{
				{
					Store: &pdapi.MetaStore{
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
	_, err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())
}

func TestPodControlUpdateMetaInfoGetMemberFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, podLister, _, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	pdClient := NewFakePDClient(pdControl, tc)
	pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		return nil, errors.New("failed to get member info from PD server")
	})
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		storesInfo := &pdapi.StoresInfo{
			Stores: []*pdapi.StoreInfo{
				{
					Store: &pdapi.MetaStore{
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
	_, err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())
}

func TestPodControlUpdateMetaInfoGetStoreFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, podLister, _, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	pdClient := NewFakePDClient(pdControl, tc)
	pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		membersInfo := &pdapi.MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		return nil, errors.New("failed to get store info from PD server")
	})

	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	pod.Labels[label.ComponentLabelKey] = label.TiKVLabelVal
	_, err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())
}

func TestPodControlUpdateMetaInfoUpdatePodFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	fakeClient, pdControl, podLister, _, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	pdClient := NewFakePDClient(pdControl, tc)
	pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		membersInfo := &pdapi.MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		storesInfo := &pdapi.StoresInfo{
			Stores: []*pdapi.StoreInfo{
				{
					Store: &pdapi.MetaStore{
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
	_, err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(HaveOccurred())
}

func TestPodControlUpdateMetaInfoConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	oldPod := newPod(tc)
	oldPod.Labels = nil
	fakeClient, pdControl, podLister, podIndexer, recorder := newFakeClientRecorderAndPDControl()
	podIndexer.Add(oldPod)
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	pdClient := NewFakePDClient(pdControl, tc)
	pdClient.AddReaction(pdapi.GetClusterActionType, func(action *pdapi.Action) (interface{}, error) {
		cluster := &metapb.Cluster{
			Id: 222,
		}
		return cluster, nil
	})
	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		membersInfo := &pdapi.MembersInfo{
			Members: []*pdpb.Member{
				{
					MemberId: 111,
				},
			},
		}
		return membersInfo, nil
	})
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		storesInfo := &pdapi.StoresInfo{
			Stores: []*pdapi.StoreInfo{
				{
					Store: &pdapi.MetaStore{
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
			return true, oldPod, apierrors.NewConflict(action.GetResource().GroupResource(), pod.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	updatePod, err := control.UpdateMetaInfo(tc, pod)
	g.Expect(err).To(Succeed())
	g.Expect(updatePod.Labels[label.StoreIDLabelKey]).To(Equal("333"))
	g.Expect(updatePod.Labels[label.ClusterIDLabelKey]).To(Equal("222"))
}

func TestPodControlUpdatePod(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	pod.Annotations = map[string]string{"a": "b"}
	pod.Labels = map[string]string{"a": "b"}
	fakeClient, pdControl, podLister, _, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		return true, update.GetObject(), nil
	})
	updatePod, err := control.UpdatePod(tc, pod)
	g.Expect(err).To(Succeed())
	g.Expect(updatePod.Annotations["a"]).To(Equal("b"))
	g.Expect(updatePod.Labels["a"]).To(Equal("b"))
}

func TestPodControlUpdatePodConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pod := newPod(tc)
	pod.Annotations = map[string]string{"a": "b"}
	pod.Labels = map[string]string{"a": "b"}
	oldPod := newPod(tc)
	fakeClient, pdControl, podLister, podIndexer, recorder := newFakeClientRecorderAndPDControl()
	control := NewRealPodControl(fakeClient, pdControl, podLister, recorder)
	err := podIndexer.Add(oldPod)
	g.Expect(err).To(Succeed())
	conflict := false
	fakeClient.AddReactor("update", "pods", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, oldPod, apierrors.NewConflict(action.GetResource().GroupResource(), pod.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	updatePod, err := control.UpdatePod(tc, pod)
	g.Expect(err).To(Succeed())
	g.Expect(updatePod.Annotations["a"]).To(Equal("b"))
	g.Expect(updatePod.Labels["a"]).To(Equal("b"))
}

func newFakeClientRecorderAndPDControl() (*fake.Clientset, *pdapi.FakePDControl, corelisters.PodLister, cache.Indexer, *record.FakeRecorder) {
	fakeClient := &fake.Clientset{}
	kubeCli := kubefake.NewSimpleClientset()
	pdControl := pdapi.NewFakePDControl(kubeCli)
	recorder := record.NewFakeRecorder(10)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	podInformer := kubeInformerFactory.Core().V1().Pods()
	return fakeClient, pdControl, podInformer.Lister(), podInformer.Informer().GetIndexer(), recorder
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
