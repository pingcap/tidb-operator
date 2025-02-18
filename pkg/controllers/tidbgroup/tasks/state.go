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
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	dbg     *v1alpha1.TiDBGroup
	dbs     []*v1alpha1.TiDB

	updateRevision  string
	currentRevision string
	collisionCount  int32
}

type State interface {
	common.TiDBGroupStateInitializer
	common.TiDBSliceStateInitializer
	common.RevisionStateInitializer[*runtime.TiDBGroup]

	common.TiDBGroupState
	common.ClusterState
	common.TiDBSliceState
	common.RevisionState

	common.GroupState[*runtime.TiDBGroup]

	common.ContextClusterNewer[*v1alpha1.TiDBGroup]

	common.InstanceSliceState[*runtime.TiDB]
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) Object() *v1alpha1.TiDBGroup {
	return s.dbg
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

func (s *state) SetCluster(cluster *v1alpha1.Cluster) {
	s.cluster = cluster
}

func (s *state) TiDBGroupInitializer() common.TiDBGroupInitializer {
	return common.NewResource(func(dbg *v1alpha1.TiDBGroup) { s.dbg = dbg }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
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
