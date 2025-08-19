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

	cluster            *v1alpha1.Cluster
	rwg                *v1alpha1.ReplicationWorkerGroup
	replicationWorkers []*v1alpha1.ReplicationWorker

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool
}

type State interface {
	common.ContextObjectNewer[*v1alpha1.ReplicationWorkerGroup]
	common.ContextClusterNewer[*v1alpha1.ReplicationWorkerGroup]
	common.ContextSliceNewer[*v1alpha1.ReplicationWorkerGroup, *v1alpha1.ReplicationWorker]
	common.ObjectState[*v1alpha1.ReplicationWorkerGroup]

	common.RevisionStateInitializer[*runtime.ReplicationWorkerGroup]

	common.ClusterState
	common.InstanceSliceState[*runtime.ReplicationWorker]
	common.SliceState[*v1alpha1.ReplicationWorker]
	common.RevisionState

	common.GroupState[*runtime.ReplicationWorkerGroup]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.ReplicationWorkerGroup]
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.ReplicationWorkerGroup {
	return s.rwg
}

func (s *state) SetObject(rwg *v1alpha1.ReplicationWorkerGroup) {
	s.rwg = rwg
}

func (s *state) ReplicationWorkerGroup() *v1alpha1.ReplicationWorkerGroup { // Added ReplicationWorkerGroup() method
	return s.rwg
}

func (s *state) Group() *runtime.ReplicationWorkerGroup {
	return runtime.FromReplicationWorkerGroup(s.rwg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) ReplicationWorkerSlice() []*v1alpha1.ReplicationWorker {
	return s.replicationWorkers
}

func (s *state) Slice() []*runtime.ReplicationWorker {
	return runtime.FromReplicationWorkerSlice(s.replicationWorkers)
}

func (s *state) InstanceSlice() []*v1alpha1.ReplicationWorker {
	return s.replicationWorkers
}

func (s *state) SetInstanceSlice(schedulers []*v1alpha1.ReplicationWorker) {
	s.replicationWorkers = schedulers
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

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.ReplicationWorkerGroup] {
	return common.NewRevision[*runtime.ReplicationWorkerGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.rwg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.rwg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.ReplicationWorkerGroup](func() *runtime.ReplicationWorkerGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentReplicationWorker,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.rwg.Name,
		}
	})
}
