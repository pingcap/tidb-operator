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

package statefulset

import (
	"bytes"
	"testing"

	asapps "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

type testcase struct {
	name        string
	sts         *apps.StatefulSet
	tc          *v1alpha1.TidbCluster
	operation   admission.Operation
	wantAllowed bool
}

var (
	ownerTCName    = "foo"
	validOwnerRefs = []metav1.OwnerReference{
		{
			APIVersion: "pingcap.com/v1alpha1",
			Kind:       "TidbCluster",
			Name:       ownerTCName,
			Controller: pointer.BoolPtr(true),
		},
	}
	tests = []testcase{
		{
			name:        "non-update operation",
			operation:   admission.Delete,
			wantAllowed: true,
		},
		{
			name:      "not tidb or tikv",
			operation: admission.Update,
			sts: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{},
			},
			wantAllowed: true,
		},
		{
			name:      "not owned by tidb cluster",
			operation: admission.Update,
			sts: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "tikv",
					},
				},
			},
			wantAllowed: true,
		},
		{
			name:      "owned by tidb cluster but the cluster does not exist",
			operation: admission.Update,
			sts: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "tikv",
					},
					OwnerReferences: validOwnerRefs,
				},
			},
			wantAllowed: false,
		},
		{
			name:      "partition does not exist",
			operation: admission.Update,
			sts: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "tikv",
					},
					OwnerReferences: validOwnerRefs,
				},
			},
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        ownerTCName,
					Namespace:   v1.NamespaceDefault,
					Annotations: nil,
				},
			},
			wantAllowed: true,
		},
		{
			name:      "partition is invald",
			operation: admission.Update,
			sts: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "tikv",
					},
					OwnerReferences: validOwnerRefs,
				},
			},
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ownerTCName,
					Namespace: v1.NamespaceDefault,
					Annotations: map[string]string{
						"tidb.pingcap.com/tikv-partition": "invalid",
					},
				},
			},
			wantAllowed: false,
		},
		{
			name:      "partition allow",
			operation: admission.Update,
			sts: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "tikv",
					},
					OwnerReferences: validOwnerRefs,
				},
				Spec: apps.StatefulSetSpec{
					UpdateStrategy: apps.StatefulSetUpdateStrategy{
						Type: apps.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
							Partition: pointer.Int32Ptr(1),
						},
					},
				},
			},
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ownerTCName,
					Namespace: v1.NamespaceDefault,
					Annotations: map[string]string{
						"tidb.pingcap.com/tikv-partition": "1",
					},
				},
			},
			wantAllowed: true,
		},
		{
			name:      "partition deny",
			operation: admission.Update,
			sts: &apps.StatefulSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/component": "tikv",
					},
					OwnerReferences: validOwnerRefs,
				},
				Spec: apps.StatefulSetSpec{
					UpdateStrategy: apps.StatefulSetUpdateStrategy{
						Type: apps.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
							Partition: pointer.Int32Ptr(0),
						},
					},
				},
			},
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ownerTCName,
					Namespace: v1.NamespaceDefault,
					Annotations: map[string]string{
						"tidb.pingcap.com/tikv-partition": "1",
					},
				},
			},
			wantAllowed: false,
		},
	}
)

func runTest(t *testing.T, tt testcase, asts bool) {
	jsonInfo, ok := runtime.SerializerInfoForMediaType(util.Codecs.SupportedMediaTypes(), runtime.ContentTypeJSON)
	if !ok {
		t.Fatalf("unable to locate encoder -- %q is not a supported media type", runtime.ContentTypeJSON)
	}

	cli := fake.NewSimpleClientset()
	ac := NewStatefulSetAdmissionControl(cli)
	ar := &admission.AdmissionRequest{
		Name:      "foo",
		Namespace: v1.NamespaceDefault,
		Operation: tt.operation,
	}
	if tt.sts != nil {
		buf := bytes.Buffer{}
		if asts {
			encoder := util.Codecs.EncoderForVersion(jsonInfo.Serializer, asapps.SchemeGroupVersion)
			sts, err := helper.FromBuiltinStatefulSet(tt.sts)
			if err != nil {
				t.Fatal(err)
			}
			if err := encoder.Encode(sts, &buf); err != nil {
				t.Fatal(err)
			}
		} else {
			encoder := util.Codecs.EncoderForVersion(jsonInfo.Serializer, apps.SchemeGroupVersion)
			if err := encoder.Encode(tt.sts, &buf); err != nil {
				t.Fatal(err)
			}
		}
		ar.Object = runtime.RawExtension{
			Raw: buf.Bytes(),
		}
	}
	if tt.tc != nil {
		cli.PingcapV1alpha1().TidbClusters(tt.tc.Namespace).Create(tt.tc)
	}
	resp := ac.AdmitStatefulSets(ar)
	if resp.Allowed != tt.wantAllowed {
		t.Errorf("want allowed %v, got %v", tt.wantAllowed, resp.Allowed)
	}
}

func TestStatefulSetAdmissionControl(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTest(t, tt, false)
		})
	}
}

func TestStatefulSetAdmissionControl_ASTS(t *testing.T) {
	saved := features.DefaultFeatureGate.String()
	features.DefaultFeatureGate.Set("AdvancedStatefulSet=true")
	defer features.DefaultFeatureGate.Set(saved) // reset features on exit

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTest(t, tt, true)
		})
	}
}
