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
	TiFlash      v1alpha1.TiFlash
	TiFlashGroup v1alpha1.TiFlashGroup
)

func FromTiFlash(f *v1alpha1.TiFlash) *TiFlash {
	return (*TiFlash)(f)
}

func FromTiFlashSlice(fs []*v1alpha1.TiFlash) []*TiFlash {
	return *(*[]*TiFlash)(unsafe.Pointer(&fs))
}

var _ instance = &TiFlash{}

func (f *TiFlash) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiFlash)(f).DeepCopyObject()
}

func (f *TiFlash) To() client.Object {
	return (*v1alpha1.TiFlash)(f)
}

func (f *TiFlash) GetTopology() v1alpha1.Topology {
	return f.Spec.Topology
}

func (f *TiFlash) SetTopology(t v1alpha1.Topology) {
	f.Spec.Topology = t
}

func (f *TiFlash) GetUpdateRevision() string {
	if f.Labels == nil {
		return ""
	}
	return f.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
}

func (f *TiFlash) IsHealthy() bool {
	return meta.IsStatusConditionTrue(f.Status.Conditions, v1alpha1.TiFlashCondHealth)
}

func (f *TiFlash) IsUpToDate() bool {
	return f.Status.ObservedGeneration == f.GetGeneration() && f.GetUpdateRevision() == f.Status.CurrentRevision
}

func (f *TiFlash) Conditions() []metav1.Condition {
	return f.Status.Conditions
}

func (f *TiFlash) Cluster() string {
	return f.Spec.Cluster.Name
}

func (*TiFlash) Component() string {
	return v1alpha1.LabelValComponentTiFlash
}
