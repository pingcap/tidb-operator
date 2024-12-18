// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"unsafe"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

type (
	TiKV      v1alpha1.TiKV
	TiKVGroup v1alpha1.TiKVGroup
)

func FromTiKV(kv *v1alpha1.TiKV) *TiKV {
	return (*TiKV)(kv)
}

func FromTiKVSlice(kvs []*v1alpha1.TiKV) []*TiKV {
	return *(*[]*TiKV)(unsafe.Pointer(&kvs))
}

var _ instance = &TiKV{}

func (kv *TiKV) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiKV)(kv).DeepCopyObject()
}

func (kv *TiKV) To() client.Object {
	return (*v1alpha1.TiKV)(kv)
}

func (kv *TiKV) GetTopology() v1alpha1.Topology {
	return kv.Spec.Topology
}

func (kv *TiKV) SetTopology(t v1alpha1.Topology) {
	kv.Spec.Topology = t
}

func (kv *TiKV) GetUpdateRevision() string {
	if kv.Labels == nil {
		return ""
	}
	return kv.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
}

func (kv *TiKV) IsHealthy() bool {
	return meta.IsStatusConditionTrue(kv.Status.Conditions, v1alpha1.TiKVCondHealth)
}

func (kv *TiKV) IsUpToDate() bool {
	return kv.Status.ObservedGeneration == kv.GetGeneration() && kv.GetUpdateRevision() == kv.Status.CurrentRevision
}

func (kv *TiKV) Conditions() []metav1.Condition {
	return kv.Status.Conditions
}

func (kv *TiKV) Cluster() string {
	return kv.Spec.Cluster.Name
}

func (*TiKV) Component() string {
	return v1alpha1.LabelValComponentTiKV
}
