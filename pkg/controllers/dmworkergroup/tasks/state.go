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

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	dwg     *v1alpha1.DMWorkerGroup
	dws     []*v1alpha1.DMWorker

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool

	stateutil.IFeatureGates
}

type State interface {
	common.DMWorkerSliceStateInitializer
	common.RevisionStateInitializer[*runtime.DMWorkerGroup]

	common.DMWorkerGroupState
	common.ClusterState
	common.DMWorkerSliceState
	common.RevisionState

	common.GroupState[*runtime.DMWorkerGroup]

	common.ContextClusterNewer[*v1alpha1.DMWorkerGroup]
	common.ContextObjectNewer[*v1alpha1.DMWorkerGroup]
	common.ContextSliceNewer[*v1alpha1.DMWorkerGroup, *v1alpha1.DMWorker]

	common.InstanceSliceState[*runtime.DMWorker]
	common.SliceState[*v1alpha1.DMWorker]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.DMWorkerGroup]

	stateutil.IFeatureGates
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.DMWorkerGroup](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.DMWorkerGroup {
	return s.dwg
}

func (s *state) SetObject(dwg *v1alpha1.DMWorkerGroup) {
	s.dwg = dwg
}

func (s *state) DMWorkerGroup() *v1alpha1.DMWorkerGroup {
	return s.dwg
}

func (s *state) Group() *runtime.DMWorkerGroup {
	return runtime.FromDMWorkerGroup(s.dwg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) DMWorkerSlice() []*v1alpha1.DMWorker {
	return s.dws
}

func (s *state) Slice() []*runtime.DMWorker {
	return runtime.FromDMWorkerSlice(s.dws)
}

func (s *state) InstanceSlice() []*v1alpha1.DMWorker {
	return s.dws
}

func (s *state) SetInstanceSlice(dws []*v1alpha1.DMWorker) {
	s.dws = dws
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

func (s *state) DMWorkerSliceInitializer() common.DMWorkerSliceInitializer {
	return common.NewResourceSlice(func(dws []*v1alpha1.DMWorker) { s.dws = dws }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.DMWorkerGroup] {
	return common.NewRevision[*runtime.DMWorkerGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.dwg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.dwg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.DMWorkerGroup](func() *runtime.DMWorkerGroup {
			return runtime.FromDMWorkerGroup(s.dwg)
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMWorker,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.dwg.Name,
		}
	})
}
