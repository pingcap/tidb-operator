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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

type (
	TiKV      v1alpha1.TiKV
	TiKVGroup v1alpha1.TiKVGroup
)

type TiKVTuple struct{}

var _ InstanceTuple[*v1alpha1.TiKV, *TiKV] = TiKVTuple{}

func (TiKVTuple) From(t *v1alpha1.TiKV) *TiKV {
	return FromTiKV(t)
}

func (TiKVTuple) FromSlice(t []*v1alpha1.TiKV) []*TiKV {
	return FromTiKVSlice(t)
}

func (TiKVTuple) To(t *TiKV) *v1alpha1.TiKV {
	return ToTiKV(t)
}

func (TiKVTuple) ToSlice(t []*TiKV) []*v1alpha1.TiKV {
	return ToTiKVSlice(t)
}

type TiKVGroupTuple struct{}

var _ GroupTuple[*v1alpha1.TiKVGroup, *TiKVGroup] = TiKVGroupTuple{}

func (TiKVGroupTuple) From(t *v1alpha1.TiKVGroup) *TiKVGroup {
	return FromTiKVGroup(t)
}

func (TiKVGroupTuple) FromSlice(t []*v1alpha1.TiKVGroup) []*TiKVGroup {
	return FromTiKVGroupSlice(t)
}

func (TiKVGroupTuple) To(t *TiKVGroup) *v1alpha1.TiKVGroup {
	return ToTiKVGroup(t)
}

func (TiKVGroupTuple) ToSlice(t []*TiKVGroup) []*v1alpha1.TiKVGroup {
	return ToTiKVGroupSlice(t)
}

func FromTiKV(kv *v1alpha1.TiKV) *TiKV {
	return (*TiKV)(kv)
}

func ToTiKV(kv *TiKV) *v1alpha1.TiKV {
	return (*v1alpha1.TiKV)(kv)
}

func FromTiKVSlice(kvs []*v1alpha1.TiKV) []*TiKV {
	return *(*[]*TiKV)(unsafe.Pointer(&kvs))
}

func ToTiKVSlice(kvs []*TiKV) []*v1alpha1.TiKV {
	return *(*[]*v1alpha1.TiKV)(unsafe.Pointer(&kvs))
}

func FromTiKVGroup(kvg *v1alpha1.TiKVGroup) *TiKVGroup {
	return (*TiKVGroup)(kvg)
}

func ToTiKVGroup(kvg *TiKVGroup) *v1alpha1.TiKVGroup {
	return (*v1alpha1.TiKVGroup)(kvg)
}

func FromTiKVGroupSlice(kvgs []*v1alpha1.TiKVGroup) []*TiKVGroup {
	return *(*[]*TiKVGroup)(unsafe.Pointer(&kvgs))
}

func ToTiKVGroupSlice(kvgs []*TiKVGroup) []*v1alpha1.TiKVGroup {
	return *(*[]*v1alpha1.TiKVGroup)(unsafe.Pointer(&kvgs))
}

var _ Instance = &TiKV{}

func (kv *TiKV) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiKV)(kv).DeepCopyObject()
}

func (kv *TiKV) To() *v1alpha1.TiKV {
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

func (kv *TiKV) CurrentRevision() string {
	return kv.Status.CurrentRevision
}

func (kv *TiKV) SetCurrentRevision(rev string) {
	kv.Status.CurrentRevision = rev
}

func (kv *TiKV) IsReady() bool {
	return meta.IsStatusConditionTrue(kv.Status.Conditions, v1alpha1.CondReady)
}

func (kv *TiKV) IsUpToDate() bool {
	return kv.Status.ObservedGeneration == kv.GetGeneration() && kv.GetUpdateRevision() == kv.Status.CurrentRevision
}

func (kv *TiKV) Conditions() []metav1.Condition {
	return kv.Status.Conditions
}

func (kv *TiKV) SetConditions(conds []metav1.Condition) {
	kv.Status.Conditions = conds
}

func (kv *TiKV) ObservedGeneration() int64 {
	return kv.Status.ObservedGeneration
}

func (kv *TiKV) SetObservedGeneration(g int64) {
	kv.Status.ObservedGeneration = g
}

func (kv *TiKV) SetCluster(cluster string) {
	kv.Spec.Cluster.Name = cluster
}

func (kv *TiKV) Cluster() string {
	return kv.Spec.Cluster.Name
}

func (*TiKV) Component() string {
	return v1alpha1.LabelValComponentTiKV
}

var _ Group = &TiKVGroup{}

func (kvg *TiKVGroup) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiKVGroup)(kvg)
}

func (kvg *TiKVGroup) To() *v1alpha1.TiKVGroup {
	return ToTiKVGroup(kvg)
}

func (kvg *TiKVGroup) SetReplicas(replicas int32) {
	kvg.Spec.Replicas = &replicas
}

func (kvg *TiKVGroup) Replicas() int32 {
	if kvg.Spec.Replicas == nil {
		return 1
	}
	return *kvg.Spec.Replicas
}

func (kvg *TiKVGroup) SetVersion(version string) {
	kvg.Spec.Template.Spec.Version = version
}

func (kvg *TiKVGroup) Version() string {
	return kvg.Spec.Template.Spec.Version
}

func (kvg *TiKVGroup) SetCluster(cluster string) {
	kvg.Spec.Cluster.Name = cluster
}

func (kvg *TiKVGroup) Cluster() string {
	return kvg.Spec.Cluster.Name
}

func (*TiKVGroup) Component() string {
	return v1alpha1.LabelValComponentTiKV
}

func (kvg *TiKVGroup) Conditions() []metav1.Condition {
	return kvg.Status.Conditions
}

func (kvg *TiKVGroup) SetConditions(conds []metav1.Condition) {
	kvg.Status.Conditions = conds
}

func (kvg *TiKVGroup) ObservedGeneration() int64 {
	return kvg.Status.ObservedGeneration
}

func (kvg *TiKVGroup) SetObservedGeneration(g int64) {
	kvg.Status.ObservedGeneration = g
}

func (kvg *TiKVGroup) SetStatusVersion(version string) {
	kvg.Status.Version = version
}

func (kvg *TiKVGroup) StatusVersion() string {
	return kvg.Status.Version
}

func (kvg *TiKVGroup) SetStatusReplicas(replicas, ready, update, current int32) {
	kvg.Status.Replicas = replicas
	kvg.Status.ReadyReplicas = ready
	kvg.Status.UpdatedReplicas = update
	kvg.Status.CurrentReplicas = current
}

func (kvg *TiKVGroup) StatusReplicas() (replicas, ready, update, current int32) {
	return kvg.Status.Replicas,
		kvg.Status.ReadyReplicas,
		kvg.Status.UpdatedReplicas,
		kvg.Status.CurrentReplicas
}

func (kvg *TiKVGroup) SetStatusRevision(update, current string, collisionCount *int32) {
	kvg.Status.UpdateRevision = update
	kvg.Status.CurrentRevision = current
	kvg.Status.CollisionCount = collisionCount
}

func (kvg *TiKVGroup) StatusRevision() (update, current string, collisionCount *int32) {
	return kvg.Status.UpdateRevision,
		kvg.Status.CurrentRevision,
		kvg.Status.CollisionCount
}
