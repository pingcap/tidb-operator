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
	"bytes"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	saved := features.DefaultFeatureGate.String()
	features.DefaultFeatureGate.Set("AutoScaling=true")
	defer features.DefaultFeatureGate.Set(saved) // reset features on exit

	type testcase struct {
		name        string
		operation   admission.Operation
		pod         *corev1.Pod
		tc          *v1alpha1.TidbCluster
		cm          *corev1.ConfigMap
		wantAllowed bool
	}

	tests := []testcase{
		{
			name:        "delete non-tidb pod",
			operation:   admission.Delete,
			pod:         &corev1.Pod{},
			wantAllowed: true,
		},
		{
			name:      "delete tidb pod",
			operation: admission.Delete,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						label.ComponentLabelKey: label.TiDBLabelVal,
					},
				},
			},
			wantAllowed: true,
		},
		{
			name:        "create non-tikv pod",
			operation:   admission.Delete,
			pod:         &corev1.Pod{},
			wantAllowed: true,
		},
		{
			name:      "create tikv pod but tidb cluster does not exist",
			operation: admission.Create,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						label.ManagedByLabelKey: label.TiDBOperator,
						label.ComponentLabelKey: label.TiKVLabelVal,
						label.InstanceLabelKey:  "tc",
					},
				},
			},
			wantAllowed: true,
		},
		{
			name:      "create tikv pod and patch for auto scaled pods",
			operation: admission.Create,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						label.ManagedByLabelKey: label.TiDBOperator,
						label.ComponentLabelKey: label.TiKVLabelVal,
						label.InstanceLabelKey:  "tc",
					},
					Name: "tc-tikv-3",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "tikv",
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "startup-script",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "tc-tikv-cfg",
									},
								},
							},
						},
					},
				},
			},
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "tc",
					Annotations: map[string]string{
						"tikv.tidb.pingcap.com/scale-out-ordinals": "[3]",
					},
				},
			},
			cm: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "tc-tikv-cfg",
				},
				Data: map[string]string{
					"config-file": ``,
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
			if tt.tc != nil {
				cli.PingcapV1alpha1().TidbClusters(tt.tc.Namespace).Create(tt.tc)
			}
			kubeCli := kubefake.NewSimpleClientset()
			if tt.cm != nil {
				kubeCli.CoreV1().ConfigMaps(tt.cm.Namespace).Create(tt.cm)
			}
			podAdmissionControl := newPodAdmissionControl(nil, kubeCli, cli)
			ar := &admission.AdmissionRequest{
				Name:      "foo",
				Namespace: namespace,
				Operation: tt.operation,
			}
			buf := bytes.Buffer{}
			if err := encoder.Encode(tt.pod, &buf); err != nil {
				t.Fatal(err)
			}
			ar.Object = runtime.RawExtension{
				Raw: buf.Bytes(),
			}
			resp := podAdmissionControl.Admit(ar)
			if resp.Allowed != tt.wantAllowed {
				t.Errorf("want %v, got %v", tt.wantAllowed, resp.Allowed)
			}
		})
	}
}

func newPodAdmissionControl(serviceAccount []string, kubeCli kubernetes.Interface, cli versioned.Interface) *PodAdmissionControl {
	ah := NewPodAdmissionControl(serviceAccount, time.Minute)
	ah.initialize(cli, kubeCli, pdapi.NewFakePDControl(kubeCli), record.NewFakeRecorder(10), wait.NeverStop)
	return ah
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
			PodName:     member.TikvPodName(tcName, int32(i)),
			LeaderCount: 1,
			State:       v1alpha1.TiKVStateUp,
		}
	}
	for i := 0; int32(i) < pdReplicas; i++ {
		tc.Status.PD.Members[member.PdPodName(tcName, int32(i))] = v1alpha1.PDMember{
			Health: true,
			Name:   member.PdPodName(tcName, int32(i)),
		}
	}
	return tc
}

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
