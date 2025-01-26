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
	TiDB      v1alpha1.TiDB
	TiDBGroup v1alpha1.TiDBGroup
)

type TiDBTuple struct{}

var _ InstanceTuple[*v1alpha1.TiDB, *TiDB] = TiDBTuple{}

func (TiDBTuple) From(t *v1alpha1.TiDB) *TiDB {
	return FromTiDB(t)
}

func (TiDBTuple) FromSlice(t []*v1alpha1.TiDB) []*TiDB {
	return FromTiDBSlice(t)
}

func (TiDBTuple) To(t *TiDB) *v1alpha1.TiDB {
	return ToTiDB(t)
}

func (TiDBTuple) ToSlice(t []*TiDB) []*v1alpha1.TiDB {
	return ToTiDBSlice(t)
}

type TiDBGroupTuple struct{}

var _ GroupTuple[*v1alpha1.TiDBGroup, *TiDBGroup] = TiDBGroupTuple{}

func (TiDBGroupTuple) From(t *v1alpha1.TiDBGroup) *TiDBGroup {
	return FromTiDBGroup(t)
}

func (TiDBGroupTuple) FromSlice(t []*v1alpha1.TiDBGroup) []*TiDBGroup {
	return FromTiDBGroupSlice(t)
}

func (TiDBGroupTuple) To(t *TiDBGroup) *v1alpha1.TiDBGroup {
	return ToTiDBGroup(t)
}

func (TiDBGroupTuple) ToSlice(t []*TiDBGroup) []*v1alpha1.TiDBGroup {
	return ToTiDBGroupSlice(t)
}

func FromTiDB(db *v1alpha1.TiDB) *TiDB {
	return (*TiDB)(db)
}

func ToTiDB(db *TiDB) *v1alpha1.TiDB {
	return (*v1alpha1.TiDB)(db)
}

func FromTiDBSlice(dbs []*v1alpha1.TiDB) []*TiDB {
	return *(*[]*TiDB)(unsafe.Pointer(&dbs))
}

func ToTiDBSlice(dbs []*TiDB) []*v1alpha1.TiDB {
	return *(*[]*v1alpha1.TiDB)(unsafe.Pointer(&dbs))
}

func FromTiDBGroup(dbg *v1alpha1.TiDBGroup) *TiDBGroup {
	return (*TiDBGroup)(dbg)
}

func ToTiDBGroup(dbg *TiDBGroup) *v1alpha1.TiDBGroup {
	return (*v1alpha1.TiDBGroup)(dbg)
}

func FromTiDBGroupSlice(dbgs []*v1alpha1.TiDBGroup) []*TiDBGroup {
	return *(*[]*TiDBGroup)(unsafe.Pointer(&dbgs))
}

func ToTiDBGroupSlice(dbgs []*TiDBGroup) []*v1alpha1.TiDBGroup {
	return *(*[]*v1alpha1.TiDBGroup)(unsafe.Pointer(&dbgs))
}

var _ Instance = &TiDB{}

func (db *TiDB) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiDB)(db).DeepCopyObject()
}

func (db *TiDB) To() *v1alpha1.TiDB {
	return (*v1alpha1.TiDB)(db)
}

func (db *TiDB) GetTopology() v1alpha1.Topology {
	return db.Spec.Topology
}

func (db *TiDB) SetTopology(t v1alpha1.Topology) {
	db.Spec.Topology = t
}

func (db *TiDB) GetUpdateRevision() string {
	if db.Labels == nil {
		return ""
	}
	return db.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
}

func (db *TiDB) CurrentRevision() string {
	return db.Status.CurrentRevision
}

func (db *TiDB) SetCurrentRevision(rev string) {
	db.Status.CurrentRevision = rev
}

func (db *TiDB) IsHealthy() bool {
	return meta.IsStatusConditionTrue(db.Status.Conditions, v1alpha1.TiKVCondHealth)
}

func (db *TiDB) IsUpToDate() bool {
	return db.Status.ObservedGeneration == db.GetGeneration() && db.GetUpdateRevision() == db.Status.CurrentRevision
}

func (db *TiDB) Conditions() []metav1.Condition {
	return db.Status.Conditions
}

func (db *TiDB) SetConditions(conds []metav1.Condition) {
	db.Status.Conditions = conds
}

func (db *TiDB) ObservedGeneration() int64 {
	return db.Status.ObservedGeneration
}

func (db *TiDB) SetObservedGeneration(g int64) {
	db.Status.ObservedGeneration = g
}

func (db *TiDB) Cluster() string {
	return db.Spec.Cluster.Name
}

func (*TiDB) Component() string {
	return v1alpha1.LabelValComponentTiDB
}

var _ Group = &TiDBGroup{}

func (dbg *TiDBGroup) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiDBGroup)(dbg)
}

func (dbg *TiDBGroup) To() *v1alpha1.TiDBGroup {
	return ToTiDBGroup(dbg)
}

func (dbg *TiDBGroup) SetReplicas(replicas int32) {
	dbg.Spec.Replicas = &replicas
}

func (dbg *TiDBGroup) Replicas() int32 {
	if dbg.Spec.Replicas == nil {
		return 1
	}
	return *dbg.Spec.Replicas
}

func (dbg *TiDBGroup) Version() string {
	return dbg.Spec.Template.Spec.Version
}

func (dbg *TiDBGroup) Cluster() string {
	return dbg.Spec.Cluster.Name
}

func (*TiDBGroup) Component() string {
	return v1alpha1.LabelValComponentTiDB
}

func (dbg *TiDBGroup) Conditions() []metav1.Condition {
	return dbg.Status.Conditions
}

func (dbg *TiDBGroup) SetConditions(conds []metav1.Condition) {
	dbg.Status.Conditions = conds
}

func (dbg *TiDBGroup) ObservedGeneration() int64 {
	return dbg.Status.ObservedGeneration
}

func (dbg *TiDBGroup) SetObservedGeneration(g int64) {
	dbg.Status.ObservedGeneration = g
}

func (dbg *TiDBGroup) SetStatusVersion(version string) {
	dbg.Status.Version = version
}

func (dbg *TiDBGroup) StatusVersion() string {
	return dbg.Status.Version
}

func (dbg *TiDBGroup) SetStatusReplicas(replicas, ready, update, current int32) {
	dbg.Status.Replicas = replicas
	dbg.Status.ReadyReplicas = ready
	dbg.Status.UpdatedReplicas = update
	dbg.Status.CurrentReplicas = current
}

func (dbg *TiDBGroup) StatusReplicas() (replicas, ready, update, current int32) {
	return dbg.Status.Replicas,
		dbg.Status.ReadyReplicas,
		dbg.Status.UpdatedReplicas,
		dbg.Status.CurrentReplicas
}

func (dbg *TiDBGroup) SetStatusRevision(update, current string, collisionCount *int32) {
	dbg.Status.UpdateRevision = update
	dbg.Status.CurrentRevision = current
	dbg.Status.CollisionCount = collisionCount
}

func (dbg *TiDBGroup) StatusRevision() (update, current string, collisionCount *int32) {
	return dbg.Status.UpdateRevision,
		dbg.Status.CurrentRevision,
		dbg.Status.CollisionCount
}
