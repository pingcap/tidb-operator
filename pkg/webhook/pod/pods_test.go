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
<<<<<<< HEAD
=======
	"bytes"
	"reflect"
>>>>>>> 1806eab... more webhook tests (#3123)
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	operatorClifake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	admission "k8s.io/api/admission/v1beta1"
<<<<<<< HEAD
=======
	appsv1 "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
>>>>>>> 1806eab... more webhook tests (#3123)
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
<<<<<<< HEAD
=======
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
>>>>>>> 1806eab... more webhook tests (#3123)
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
			PD: v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
				Replicas:         pdReplicas,
				StorageClassName: pointer.StringPtr("my-storage-class"),
			},
			TiKV: v1alpha1.TiKVSpec{
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

<<<<<<< HEAD
func newNormalPod() *corev1.Pod {
	pod := corev1.Pod{}
	pod.Name = "normalPod"
	pod.Labels = map[string]string{}
	pod.Namespace = namespace
	return &pod
=======
func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		operation   admission.Operation
		username    string
		pod         *corev1.Pod
		sts         *appsv1.StatefulSet
		tc          *v1alpha1.TidbCluster
		cm          *corev1.ConfigMap
		wantAllowed bool
	}{
		{
			name:      "unknown service account",
			operation: admission.Create,
			username:  "does-not-exist",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
				},
			},
			wantAllowed: true,
		},
		{
			name:      "create a pod that is not managed by tidb-operator",
			operation: admission.Create,
			username:  "system:serviceaccount:kube-system:statefulset-controller",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
				},
			},
			wantAllowed: true,
		},
		{
			name:      "create a pod but tidb cluster does not exist",
			operation: admission.Create,
			username:  "system:serviceaccount:kube-system:statefulset-controller",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
					Labels: map[string]string{
						label.ManagedByLabelKey: label.TiDBOperator,
						label.ComponentLabelKey: label.TiKVLabelVal,
						label.NameLabelKey:      "tidb-cluster",
						label.InstanceLabelKey:  "tc",
					},
				},
			},
			wantAllowed: false,
		},
		{
			name:      "delete a pod that is not managed by tidb-operator",
			operation: admission.Delete,
			username:  "system:serviceaccount:kube-system:statefulset-controller",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
				},
			},
			wantAllowed: true,
		},
		{
			name:      "unsupported operation",
			operation: admission.Connect,
			username:  "system:serviceaccount:kube-system:statefulset-controller",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
				},
			},
			wantAllowed: true,
		},
		{
			name:      "delete a PD pod and statefulset owner does not exist",
			operation: admission.Delete,
			username:  "system:serviceaccount:kube-system:statefulset-controller",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
					Labels: map[string]string{
						label.ManagedByLabelKey: label.TiDBOperator,
						label.ComponentLabelKey: label.PDLabelVal,
						label.InstanceLabelKey:  "tc",
					},
				},
			},
			wantAllowed: true,
		},
		{
			name:      "delete a PD pod and tc does not exist",
			operation: admission.Delete,
			username:  "system:serviceaccount:kube-system:statefulset-controller",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
					Labels: map[string]string{
						label.ManagedByLabelKey: label.TiDBOperator,
						label.ComponentLabelKey: label.PDLabelVal,
						label.InstanceLabelKey:  "tc",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "StatefulSet",
							Name: "sts",
						},
					},
				},
			},
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "sts",
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: pointer.Int32Ptr(3),
				},
				Status: appsv1.StatefulSetStatus{
					Replicas: 3,
				},
			},
			wantAllowed: true,
		},
		{
			name:      "delete a PD pod and force upgrade is enabled",
			operation: admission.Delete,
			username:  "system:serviceaccount:kube-system:statefulset-controller",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "foo",
					Labels: map[string]string{
						label.ManagedByLabelKey: label.TiDBOperator,
						label.ComponentLabelKey: label.PDLabelVal,
						label.InstanceLabelKey:  "tc",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "StatefulSet",
							Name: "sts",
						},
					},
				},
			},
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "sts",
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: pointer.Int32Ptr(3),
				},
				Status: appsv1.StatefulSetStatus{
					Replicas: 3,
				},
			},
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: corev1.NamespaceDefault,
					Name:      "tc",
					Annotations: map[string]string{
						"tidb.pingcap.com/force-upgrade": "true",
					},
				},
			},
			wantAllowed: true,
		},
	}

	jsonInfo, ok := runtime.SerializerInfoForMediaType(util.Codecs.SupportedMediaTypes(), runtime.ContentTypeJSON)
	if !ok {
		t.Fatalf("unable to locate encoder -- %q is not a supported media type", runtime.ContentTypeJSON)
	}
	encoder := util.Codecs.EncoderForVersion(jsonInfo.Serializer, corev1.SchemeGroupVersion)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset()
			kubeCli := kubefake.NewSimpleClientset()
			if tt.pod != nil {
				kubeCli.CoreV1().Pods(tt.pod.Namespace).Create(tt.pod)
			}
			if tt.sts != nil {
				kubeCli.AppsV1().StatefulSets(tt.sts.Namespace).Create(tt.sts)
			}
			if tt.tc != nil {
				cli.PingcapV1alpha1().TidbClusters(tt.tc.Namespace).Create(tt.tc)
			}
			podAdmissionControl := newPodAdmissionControl(nil, kubeCli, cli)
			ar := &admission.AdmissionRequest{
				Name:      "foo",
				Namespace: v1.NamespaceDefault,
				Operation: tt.operation,
				UserInfo: authenticationv1.UserInfo{
					Username: tt.username,
				},
			}
			buf := bytes.Buffer{}
			if err := encoder.Encode(tt.pod, &buf); err != nil {
				t.Fatal(err)
			}
			ar.Object = runtime.RawExtension{
				Raw: buf.Bytes(),
			}
			resp := podAdmissionControl.Validate(ar)
			if resp.Allowed != tt.wantAllowed {
				t.Errorf("want %v, got %v", tt.wantAllowed, resp.Allowed)
			}
		})
	}
>>>>>>> 1806eab... more webhook tests (#3123)
}

func TestValidatingResource(t *testing.T) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	w := newPodAdmissionControl(nil, kubeCli, cli)
	wantGvr := schema.GroupVersionResource{
		Group:    "admission.tidb.pingcap.com",
		Version:  "v1alpha1",
		Resource: "podvalidations",
	}
	gvr, _ := w.ValidatingResource()
	if !reflect.DeepEqual(wantGvr, gvr) {
		t.Fatalf("want: %v, got: %v", wantGvr, gvr)
	}
}

func TestMutationResource(t *testing.T) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	w := newPodAdmissionControl(nil, kubeCli, cli)
	wantGvr := schema.GroupVersionResource{
		Group:    "admission.tidb.pingcap.com",
		Version:  "v1alpha1",
		Resource: "podmutations",
	}
	gvr, _ := w.MutatingResource()
	if !reflect.DeepEqual(wantGvr, gvr) {
		t.Fatalf("want: %v, got: %v", wantGvr, gvr)
	}
}
