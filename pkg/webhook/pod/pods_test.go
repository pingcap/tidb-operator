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

package pod

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	operatorClifake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	admission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

const (
	tcName              = "tc"
	upgradeInstanceName = "upgrader"
	namespace           = "namespace"
	pdReplicas          = int32(3)
	tikvReplicas        = int32(3)
)

func TestAdmitPod(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		isDelete bool
		isPD     bool
		isTiKV   bool
		isTiDB   bool
		expectFn func(g *GomegaWithT, response *admission.AdmissionResponse)
	}

	testFn := func(test *testcase) {
		t.Log(test.name)

		podAdmissionControl, _, _, podIndexer, _, _ := newPodAdmissionControl()
		ar := newAdmissionReview()
		pod := newNormalPod()
		if test.isDelete {
			ar.Request.Operation = admission.Delete
		}

		if test.isPD {
			pod.Labels = map[string]string{
				label.ComponentLabelKey: label.PDLabelVal,
			}
		} else if test.isTiDB {
			pod.Labels = map[string]string{
				label.ComponentLabelKey: label.TiDBLabelVal,
			}
		} else if test.isTiKV {
			pod.Labels = map[string]string{
				label.ComponentLabelKey: label.TiKVLabelVal,
			}
		}
		podIndexer.Add(pod)

		resp := podAdmissionControl.AdmitPods(*ar)
		test.expectFn(g, resp)
	}

	tests := []testcase{
		{
			name:     "Delete Non-TiDB Pod",
			isDelete: true,
			isPD:     false,
			isTiKV:   false,
			isTiDB:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
		{
			name:     "Non-Delete Pod",
			isDelete: false,
			isPD:     false,
			isTiKV:   false,
			isTiDB:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
		{
			name:     "Delete Non-PD Pod",
			isDelete: false,
			isPD:     false,
			isTiKV:   true,
			isTiDB:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
	}

	for _, test := range tests {
		testFn(&test)
	}
}

func newAdmissionReview() *admission.AdmissionReview {
	ar := admission.AdmissionReview{}
	request := admission.AdmissionRequest{}
	request.Name = "pod"
	request.Namespace = namespace
	request.Operation = admission.Update
	ar.Request = &request
	return &ar
}

func newPodAdmissionControl() (*PodAdmissionControl, *controller.FakePVCControl, cache.Indexer, cache.Indexer, cache.Indexer, cache.Indexer) {
	kubeCli := kubefake.NewSimpleClientset()
	operatorCli := operatorClifake.NewSimpleClientset()
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvcControl := controller.NewFakePVCControl(pvcInformer)
	pdControl := pdapi.NewFakePDControl(kubeCli)
	podInformer := kubeInformerFactory.Core().V1().Pods()
	informer := informers.NewSharedInformerFactory(operatorCli, 0)
	stsInformer := kubeInformerFactory.Apps().V1().StatefulSets()

	return &PodAdmissionControl{
			kubeCli:     kubeCli,
			operatorCli: operatorCli,
			pvcControl:  pvcControl,
			pdControl:   pdControl,
			podLister:   podInformer.Lister(),
			tcLister:    informer.Pingcap().V1alpha1().TidbClusters().Lister(),
			stsLister:   stsInformer.Lister(),
		}, pvcControl,
		pvcInformer.Informer().GetIndexer(),
		podInformer.Informer().GetIndexer(),
		informer.Pingcap().V1alpha1().TidbClusters().Informer().GetIndexer(),
		stsInformer.Informer().GetIndexer()
}

func newTidbClusterForPodAdmissionControl() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tcName,
			Namespace: namespace,
			UID:       types.UID(tcName),
			Labels:    label.New().Instance(upgradeInstanceName),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         pdReplicas,
				StorageClassName: "my-storage-class",
			},
			TiKV: v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
				Replicas:         tikvReplicas,
				StorageClassName: "my-storage-class",
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			TiKV: v1alpha1.TiKVStatus{
				Synced: true,
				Phase:  v1alpha1.NormalPhase,
				Stores: map[string]v1alpha1.TiKVStore{
					"0": {
						PodName:     memberUtils.TikvPodName(tcName, 0),
						LeaderCount: 1,
						State:       v1alpha1.TiKVStateUp,
					},
					"1": {
						PodName:     memberUtils.TikvPodName(tcName, 1),
						LeaderCount: 1,
						State:       v1alpha1.TiKVStateUp,
					},
					"2": {
						PodName:     memberUtils.TikvPodName(tcName, 2),
						LeaderCount: 1,
						State:       v1alpha1.TiKVStateUp,
					},
				},
			},
		},
	}
}

func newNormalPod() *corev1.Pod {
	pod := corev1.Pod{}
	pod.Name = "normalPod"
	pod.Labels = map[string]string{}
	pod.Namespace = namespace
	return &pod
}
