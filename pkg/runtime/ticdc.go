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
	TiCDC      v1alpha1.TiCDC
	TiCDCGroup v1alpha1.TiCDCGroup
)

type TiCDCTuple struct{}

var _ InstanceTuple[*v1alpha1.TiCDC, *TiCDC] = TiCDCTuple{}

func (TiCDCTuple) From(t *v1alpha1.TiCDC) *TiCDC {
	return FromTiCDC(t)
}

func (TiCDCTuple) FromSlice(t []*v1alpha1.TiCDC) []*TiCDC {
	return FromTiCDCSlice(t)
}

func (TiCDCTuple) To(t *TiCDC) *v1alpha1.TiCDC {
	return ToTiCDC(t)
}

func (TiCDCTuple) ToSlice(t []*TiCDC) []*v1alpha1.TiCDC {
	return ToTiCDCSlice(t)
}

type TiCDCGroupTuple struct{}

var _ GroupTuple[*v1alpha1.TiCDCGroup, *TiCDCGroup] = TiCDCGroupTuple{}

func (TiCDCGroupTuple) From(t *v1alpha1.TiCDCGroup) *TiCDCGroup {
	return FromTiCDCGroup(t)
}

func (TiCDCGroupTuple) FromSlice(t []*v1alpha1.TiCDCGroup) []*TiCDCGroup {
	return FromTiCDCGroupSlice(t)
}

func (TiCDCGroupTuple) To(t *TiCDCGroup) *v1alpha1.TiCDCGroup {
	return ToTiCDCGroup(t)
}

func (TiCDCGroupTuple) ToSlice(t []*TiCDCGroup) []*v1alpha1.TiCDCGroup {
	return ToTiCDCGroupSlice(t)
}

func FromTiCDC(cdc *v1alpha1.TiCDC) *TiCDC {
	return (*TiCDC)(cdc)
}

func ToTiCDC(cdc *TiCDC) *v1alpha1.TiCDC {
	return (*v1alpha1.TiCDC)(cdc)
}

func FromTiCDCSlice(cdcs []*v1alpha1.TiCDC) []*TiCDC {
	return *(*[]*TiCDC)(unsafe.Pointer(&cdcs))
}

func ToTiCDCSlice(cdcs []*TiCDC) []*v1alpha1.TiCDC {
	return *(*[]*v1alpha1.TiCDC)(unsafe.Pointer(&cdcs))
}

func FromTiCDCGroup(cdcg *v1alpha1.TiCDCGroup) *TiCDCGroup {
	return (*TiCDCGroup)(cdcg)
}

func ToTiCDCGroup(cdcg *TiCDCGroup) *v1alpha1.TiCDCGroup {
	return (*v1alpha1.TiCDCGroup)(cdcg)
}

func FromTiCDCGroupSlice(cdcgs []*v1alpha1.TiCDCGroup) []*TiCDCGroup {
	return *(*[]*TiCDCGroup)(unsafe.Pointer(&cdcgs))
}

func ToTiCDCGroupSlice(cdcgs []*TiCDCGroup) []*v1alpha1.TiCDCGroup {
	return *(*[]*v1alpha1.TiCDCGroup)(unsafe.Pointer(&cdcgs))
}

var _ Instance = &TiCDC{}

func (cdc *TiCDC) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiCDC)(cdc).DeepCopyObject()
}

func (cdc *TiCDC) To() *v1alpha1.TiCDC {
	return (*v1alpha1.TiCDC)(cdc)
}

func (cdc *TiCDC) GetTopology() v1alpha1.Topology {
	return cdc.Spec.Topology
}

func (cdc *TiCDC) SetTopology(t v1alpha1.Topology) {
	cdc.Spec.Topology = t
}

func (cdc *TiCDC) GetUpdateRevision() string {
	if cdc.Labels == nil {
		return ""
	}
	return cdc.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
}

func (cdc *TiCDC) CurrentRevision() string {
	return cdc.Status.CurrentRevision
}

func (cdc *TiCDC) SetCurrentRevision(rev string) {
	cdc.Status.CurrentRevision = rev
}

func (cdc *TiCDC) IsHealthy() bool {
	return meta.IsStatusConditionTrue(cdc.Status.Conditions, v1alpha1.TiCDCCondHealth)
}

func (cdc *TiCDC) IsUpToDate() bool {
	return cdc.Status.ObservedGeneration == cdc.GetGeneration() && cdc.GetUpdateRevision() == cdc.Status.CurrentRevision
}

func (cdc *TiCDC) Conditions() []metav1.Condition {
	return cdc.Status.Conditions
}

func (cdc *TiCDC) SetConditions(conds []metav1.Condition) {
	cdc.Status.Conditions = conds
}

func (cdc *TiCDC) ObservedGeneration() int64 {
	return cdc.Status.ObservedGeneration
}

func (cdc *TiCDC) SetObservedGeneration(g int64) {
	cdc.Status.ObservedGeneration = g
}

func (cdc *TiCDC) SetCluster(cluster string) {
	cdc.Spec.Cluster.Name = cluster
}

func (cdc *TiCDC) Cluster() string {
	return cdc.Spec.Cluster.Name
}

func (*TiCDC) Component() string {
	return v1alpha1.LabelValComponentTiCDC
}

var _ Group = &TiCDCGroup{}

func (cdcg *TiCDCGroup) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiCDCGroup)(cdcg)
}

func (cdcg *TiCDCGroup) To() *v1alpha1.TiCDCGroup {
	return ToTiCDCGroup(cdcg)
}

func (cdcg *TiCDCGroup) SetReplicas(replicas int32) {
	cdcg.Spec.Replicas = &replicas
}

func (cdcg *TiCDCGroup) Replicas() int32 {
	if cdcg.Spec.Replicas == nil {
		return 1
	}
	return *cdcg.Spec.Replicas
}

func (cdcg *TiCDCGroup) SetVersion(version string) {
	cdcg.Spec.Template.Spec.Version = version
}

func (cdcg *TiCDCGroup) Version() string {
	return cdcg.Spec.Template.Spec.Version
}

func (cdcg *TiCDCGroup) SetCluster(cluster string) {
	cdcg.Spec.Cluster.Name = cluster
}

func (cdcg *TiCDCGroup) Cluster() string {
	return cdcg.Spec.Cluster.Name
}

func (*TiCDCGroup) Component() string {
	return v1alpha1.LabelValComponentTiCDC
}

func (cdcg *TiCDCGroup) Conditions() []metav1.Condition {
	return cdcg.Status.Conditions
}

func (cdcg *TiCDCGroup) SetConditions(conds []metav1.Condition) {
	cdcg.Status.Conditions = conds
}

func (cdcg *TiCDCGroup) ObservedGeneration() int64 {
	return cdcg.Status.ObservedGeneration
}

func (cdcg *TiCDCGroup) SetObservedGeneration(g int64) {
	cdcg.Status.ObservedGeneration = g
}

func (cdcg *TiCDCGroup) SetStatusVersion(version string) {
	cdcg.Status.Version = version
}

func (cdcg *TiCDCGroup) StatusVersion() string {
	return cdcg.Status.Version
}

func (cdcg *TiCDCGroup) SetStatusReplicas(replicas, ready, update, current int32) {
	cdcg.Status.Replicas = replicas
	cdcg.Status.ReadyReplicas = ready
	cdcg.Status.UpdatedReplicas = update
	cdcg.Status.CurrentReplicas = current
}

func (cdcg *TiCDCGroup) StatusReplicas() (replicas, ready, update, current int32) {
	return cdcg.Status.Replicas,
		cdcg.Status.ReadyReplicas,
		cdcg.Status.UpdatedReplicas,
		cdcg.Status.CurrentReplicas
}

func (cdcg *TiCDCGroup) SetStatusRevision(update, current string, collisionCount *int32) {
	cdcg.Status.UpdateRevision = update
	cdcg.Status.CurrentRevision = current
	cdcg.Status.CollisionCount = collisionCount
}

func (cdcg *TiCDCGroup) StatusRevision() (update, current string, collisionCount *int32) {
	return cdcg.Status.UpdateRevision,
		cdcg.Status.CurrentRevision,
		cdcg.Status.CollisionCount
}
