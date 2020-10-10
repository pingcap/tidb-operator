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
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	operatorClifake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	admission "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
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

		kubeCli := kubefake.NewSimpleClientset()
		podAdmissionControl := newPodAdmissionControl(kubeCli)
		ar := newAdmissionRequest()
		pod := newNormalPod()
		if test.isDelete {
			ar.Operation = admission.Delete
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

		resp := podAdmissionControl.AdmitPods(ar)
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

func newAdmissionRequest() *admission.AdmissionRequest {
	request := admission.AdmissionRequest{}
	request.Name = "pod"
	request.Namespace = namespace
	request.Operation = admission.Update
	return &request
}

func newPodAdmissionControl(kubeCli kubernetes.Interface) *PodAdmissionControl {
	operatorCli := operatorClifake.NewSimpleClientset()
	pdControl := pdapi.NewFakePDControl(kubeCli)
	recorder := record.NewFakeRecorder(10)
	return &PodAdmissionControl{
		kubeCli:     kubeCli,
		operatorCli: operatorCli,
		pdControl:   pdControl,
		recorder:    recorder,
	}
}

func newTidbClusterForPodAdmissionControl(pdReplicas int32, tikvReplicas int32) *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{
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
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         pdReplicas,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiKV: &v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
				Replicas:         tikvReplicas,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			TiKV: v1alpha1.TiKVStatus{
				Synced: true,
				Phase:  v1alpha1.NormalPhase,
				Stores: map[string]v1alpha1.TiKVStore{},
			},
			PD: v1alpha1.PDStatus{
				Synced:  true,
				Phase:   v1alpha1.NormalPhase,
				Members: map[string]v1alpha1.PDMember{},
			},
		},
	}
	for i := 0; int32(i) < tikvReplicas; i++ {
		tc.Status.TiKV.Stores[strconv.Itoa(i)] = v1alpha1.TiKVStore{
			PodName:     memberUtils.TikvPodName(tcName, int32(i)),
			LeaderCount: 1,
			State:       v1alpha1.TiKVStateUp,
		}
	}
	for i := 0; int32(i) < pdReplicas; i++ {
		tc.Status.PD.Members[memberUtils.PdPodName(tcName, int32(i))] = v1alpha1.PDMember{
			Health: true,
			Name:   memberUtils.PdPodName(tcName, int32(i)),
		}
	}
	return tc
}

func newNormalPod() *corev1.Pod {
	pod := corev1.Pod{}
	pod.Name = "normalPod"
	pod.Labels = map[string]string{}
	pod.Namespace = namespace
	return &pod
}
