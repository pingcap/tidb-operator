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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

type (
	TiFlash      v1alpha1.TiFlash
	TiFlashGroup v1alpha1.TiFlashGroup
)

type TiFlashTuple struct{}

var _ InstanceTuple[*v1alpha1.TiFlash, *TiFlash] = TiFlashTuple{}

func (TiFlashTuple) From(t *v1alpha1.TiFlash) *TiFlash {
	return FromTiFlash(t)
}

func (TiFlashTuple) FromSlice(t []*v1alpha1.TiFlash) []*TiFlash {
	return FromTiFlashSlice(t)
}

func (TiFlashTuple) To(t *TiFlash) *v1alpha1.TiFlash {
	return ToTiFlash(t)
}

func (TiFlashTuple) ToSlice(t []*TiFlash) []*v1alpha1.TiFlash {
	return ToTiFlashSlice(t)
}

type TiFlashGroupTuple struct{}

var _ GroupTuple[*v1alpha1.TiFlashGroup, *TiFlashGroup] = TiFlashGroupTuple{}

func (TiFlashGroupTuple) From(t *v1alpha1.TiFlashGroup) *TiFlashGroup {
	return FromTiFlashGroup(t)
}

func (TiFlashGroupTuple) FromSlice(t []*v1alpha1.TiFlashGroup) []*TiFlashGroup {
	return FromTiFlashGroupSlice(t)
}

func (TiFlashGroupTuple) To(t *TiFlashGroup) *v1alpha1.TiFlashGroup {
	return ToTiFlashGroup(t)
}

func (TiFlashGroupTuple) ToSlice(t []*TiFlashGroup) []*v1alpha1.TiFlashGroup {
	return ToTiFlashGroupSlice(t)
}

func FromTiFlash(f *v1alpha1.TiFlash) *TiFlash {
	return (*TiFlash)(f)
}

func ToTiFlash(f *TiFlash) *v1alpha1.TiFlash {
	return (*v1alpha1.TiFlash)(f)
}

func FromTiFlashSlice(fs []*v1alpha1.TiFlash) []*TiFlash {
	return *(*[]*TiFlash)(unsafe.Pointer(&fs))
}

func ToTiFlashSlice(fs []*TiFlash) []*v1alpha1.TiFlash {
	return *(*[]*v1alpha1.TiFlash)(unsafe.Pointer(&fs))
}

func FromTiFlashGroup(fg *v1alpha1.TiFlashGroup) *TiFlashGroup {
	return (*TiFlashGroup)(fg)
}

func ToTiFlashGroup(fg *TiFlashGroup) *v1alpha1.TiFlashGroup {
	return (*v1alpha1.TiFlashGroup)(fg)
}

func FromTiFlashGroupSlice(fgs []*v1alpha1.TiFlashGroup) []*TiFlashGroup {
	return *(*[]*TiFlashGroup)(unsafe.Pointer(&fgs))
}

func ToTiFlashGroupSlice(fgs []*TiFlashGroup) []*v1alpha1.TiFlashGroup {
	return *(*[]*v1alpha1.TiFlashGroup)(unsafe.Pointer(&fgs))
}

var _ Instance = &TiFlash{}

func (f *TiFlash) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiFlash)(f).DeepCopyObject()
}

func (f *TiFlash) To() *v1alpha1.TiFlash {
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

func (f *TiFlash) CurrentRevision() string {
	return f.Status.CurrentRevision
}

func (f *TiFlash) SetCurrentRevision(rev string) {
	f.Status.CurrentRevision = rev
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

func (f *TiFlash) SetConditions(conds []metav1.Condition) {
	f.Status.Conditions = conds
}

func (f *TiFlash) ObservedGeneration() int64 {
	return f.Status.ObservedGeneration
}

func (f *TiFlash) SetObservedGeneration(g int64) {
	f.Status.ObservedGeneration = g
}

func (f *TiFlash) Cluster() string {
	return f.Spec.Cluster.Name
}

func (*TiFlash) Component() string {
	return v1alpha1.LabelValComponentTiFlash
}

var _ Group = &TiFlashGroup{}

func (fg *TiFlashGroup) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiFlashGroup)(fg)
}

func (fg *TiFlashGroup) To() *v1alpha1.TiFlashGroup {
	return ToTiFlashGroup(fg)
}

func (fg *TiFlashGroup) SetReplicas(replicas int32) {
	fg.Spec.Replicas = &replicas
}

func (fg *TiFlashGroup) Replicas() int32 {
	if fg.Spec.Replicas == nil {
		return 1
	}
	return *fg.Spec.Replicas
}

func (fg *TiFlashGroup) Version() string {
	return fg.Spec.Template.Spec.Version
}

func (fg *TiFlashGroup) Cluster() string {
	return fg.Spec.Cluster.Name
}

func (*TiFlashGroup) Component() string {
	return v1alpha1.LabelValComponentTiFlash
}

func (fg *TiFlashGroup) Conditions() []metav1.Condition {
	return fg.Status.Conditions
}

func (fg *TiFlashGroup) SetConditions(conds []metav1.Condition) {
	fg.Status.Conditions = conds
}

func (fg *TiFlashGroup) ObservedGeneration() int64 {
	return fg.Status.ObservedGeneration
}

func (fg *TiFlashGroup) SetObservedGeneration(g int64) {
	fg.Status.ObservedGeneration = g
}

func (fg *TiFlashGroup) SetStatusVersion(version string) {
	fg.Status.Version = version
}

func (fg *TiFlashGroup) StatusVersion() string {
	return fg.Status.Version
}

func (fg *TiFlashGroup) SetStatusReplicas(replicas, ready, update, current int32) {
	fg.Status.Replicas = replicas
	fg.Status.ReadyReplicas = ready
	fg.Status.UpdatedReplicas = update
	fg.Status.CurrentReplicas = current
}

func (fg *TiFlashGroup) StatusReplicas() (replicas, ready, update, current int32) {
	return fg.Status.Replicas,
		fg.Status.ReadyReplicas,
		fg.Status.UpdatedReplicas,
		fg.Status.CurrentReplicas
}

func (fg *TiFlashGroup) SetStatusRevision(update, current string, collisionCount *int32) {
	fg.Status.UpdateRevision = update
	fg.Status.CurrentRevision = current
	fg.Status.CollisionCount = collisionCount
}

func (fg *TiFlashGroup) StatusRevision() (update, current string, collisionCount *int32) {
	return fg.Status.UpdateRevision,
		fg.Status.CurrentRevision,
		fg.Status.CollisionCount
}
