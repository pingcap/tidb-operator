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
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
)

type ReconcileContext struct {
	State
}

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	wg      *v1alpha1.TiKVWorkerGroup
	ws      []*v1alpha1.TiKVWorker

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool

	stateutil.IFeatureGates
}

type State interface {
	common.ContextObjectNewer[*v1alpha1.TiKVWorkerGroup]
	common.ContextClusterNewer[*v1alpha1.TiKVWorkerGroup]
	common.ContextSliceNewer[*v1alpha1.TiKVWorkerGroup, *v1alpha1.TiKVWorker]

	common.RevisionStateInitializer[*runtime.TiKVWorkerGroup]

	common.ClusterState
	common.InstanceSliceState[*runtime.TiKVWorker]
	common.SliceState[*v1alpha1.TiKVWorker]
	common.RevisionState

	common.GroupState[*runtime.TiKVWorkerGroup]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.TiKVWorkerGroup]

	stateutil.IFeatureGates
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.TiKVWorkerGroup](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.TiKVWorkerGroup {
	return s.wg
}

func (s *state) SetObject(wg *v1alpha1.TiKVWorkerGroup) {
	s.wg = wg
}

func (s *state) TiKVWorkerGroup() *v1alpha1.TiKVWorkerGroup {
	return s.wg
}

func (s *state) Group() *runtime.TiKVWorkerGroup {
	return runtime.FromTiKVWorkerGroup(s.wg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TiKVWorkerSlice() []*v1alpha1.TiKVWorker {
	return s.ws
}

func (s *state) Slice() []*runtime.TiKVWorker {
	return runtime.FromTiKVWorkerSlice(s.ws)
}

func (s *state) InstanceSlice() []*v1alpha1.TiKVWorker {
	return s.ws
}

func (s *state) SetInstanceSlice(ws []*v1alpha1.TiKVWorker) {
	s.ws = ws
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

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.TiKVWorkerGroup] {
	return common.NewRevision[*runtime.TiKVWorkerGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.wg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.wg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.TiKVWorkerGroup](func() *runtime.TiKVWorkerGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKVWorker,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.wg.Name,
		}
	})
}
