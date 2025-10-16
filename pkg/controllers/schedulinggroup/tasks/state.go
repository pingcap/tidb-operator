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

	cluster    *v1alpha1.Cluster
	sg         *v1alpha1.SchedulingGroup
	schedulings []*v1alpha1.Scheduling

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool
	stateutil.IFeatureGates
}

type State interface {
	common.ContextObjectNewer[*v1alpha1.SchedulingGroup]
	common.ContextClusterNewer[*v1alpha1.SchedulingGroup]
	common.ContextSliceNewer[*v1alpha1.SchedulingGroup, *v1alpha1.Scheduling]
	common.ObjectState[*v1alpha1.SchedulingGroup]

	common.RevisionStateInitializer[*runtime.SchedulingGroup]

	common.ClusterState
	common.InstanceSliceState[*runtime.Scheduling]
	common.SliceState[*v1alpha1.Scheduling]
	common.RevisionState

	common.GroupState[*runtime.SchedulingGroup]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.SchedulingGroup]

	stateutil.IFeatureGates
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.SchedulingGroup](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.SchedulingGroup {
	return s.sg
}

func (s *state) SetObject(sg *v1alpha1.SchedulingGroup) {
	s.sg = sg
}

func (s *state) SchedulingGroup() *v1alpha1.SchedulingGroup { // Added SchedulingGroup() method
	return s.sg
}

func (s *state) Group() *runtime.SchedulingGroup {
	return runtime.FromSchedulingGroup(s.sg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) SchedulingSlice() []*v1alpha1.Scheduling {
	return s.schedulings
}

func (s *state) Slice() []*runtime.Scheduling {
	return runtime.FromSchedulingSlice(s.schedulings)
}

func (s *state) InstanceSlice() []*v1alpha1.Scheduling {
	return s.schedulings
}

func (s *state) SetInstanceSlice(schedulings []*v1alpha1.Scheduling) {
	s.schedulings = schedulings
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

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.SchedulingGroup] {
	return common.NewRevision[*runtime.SchedulingGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.sg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.sg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.SchedulingGroup](func() *runtime.SchedulingGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentScheduling,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.sg.Name,
		}
	})
}
