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

package tasks

import (
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/pkg/state"
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	dbg     *v1alpha1.TiDBGroup
	dbs     []*v1alpha1.TiDB

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool

	stateutil.IFeatureGates
}

type State interface {
	common.TiDBSliceStateInitializer
	common.RevisionStateInitializer[*runtime.TiDBGroup]

	common.TiDBGroupState
	common.ClusterState
	common.TiDBSliceState
	common.RevisionState

	common.GroupState[*runtime.TiDBGroup]

	common.ContextClusterNewer[*v1alpha1.TiDBGroup]
	common.ContextObjectNewer[*v1alpha1.TiDBGroup]
	common.ContextSliceNewer[*v1alpha1.TiDBGroup, *v1alpha1.TiDB]

	common.InstanceSliceState[*runtime.TiDB]
	common.SliceState[*v1alpha1.TiDB]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.TiDBGroup]

	stateutil.IFeatureGates
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.TiDBGroup](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.TiDBGroup {
	return s.dbg
}

func (s *state) SetObject(dbg *v1alpha1.TiDBGroup) {
	s.dbg = dbg
}

func (s *state) TiDBGroup() *v1alpha1.TiDBGroup {
	return s.dbg
}

func (s *state) Group() *runtime.TiDBGroup {
	return runtime.FromTiDBGroup(s.dbg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TiDBSlice() []*v1alpha1.TiDB {
	return s.dbs
}

func (s *state) Slice() []*runtime.TiDB {
	return runtime.FromTiDBSlice(s.dbs)
}

func (s *state) InstanceSlice() []*v1alpha1.TiDB {
	return s.dbs
}

func (s *state) SetInstanceSlice(dbs []*v1alpha1.TiDB) {
	s.dbs = dbs
}

func (s *state) SetCluster(cluster *v1alpha1.Cluster) {
	s.cluster = cluster
}

func (s *state) IsStatusChanged() bool {
	return s.statusChanged
}

func (s *state) SetStatusChanged() {
	s.statusChanged = true
}

func (s *state) TiDBSliceInitializer() common.TiDBSliceInitializer {
	return common.NewResourceSlice(func(dbs []*v1alpha1.TiDB) { s.dbs = dbs }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.TiDBGroup] {
	return common.NewRevision[*runtime.TiDBGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.dbg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.dbg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.TiDBGroup](func() *runtime.TiDBGroup {
			return s.Group()
		})).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) Revision() (update, current string, collisionCount int32) {
	return s.updateRevision, s.currentRevision, s.collisionCount
}

func (s *state) Labels() common.LabelsOption {
	return common.Lazy[map[string]string](func() map[string]string {
		return map[string]string{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.dbg.Name,
		}
	})
}
