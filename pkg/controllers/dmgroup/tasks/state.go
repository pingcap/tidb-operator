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
	dmg     *v1alpha1.DMGroup
	dms     []*v1alpha1.DM

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool

	stateutil.IFeatureGates
}

type State interface {
	common.DMSliceStateInitializer
	common.RevisionStateInitializer[*runtime.DMGroup]

	common.DMGroupState
	common.ClusterState
	common.DMSliceState
	common.RevisionState

	common.GroupState[*runtime.DMGroup]

	common.ContextClusterNewer[*v1alpha1.DMGroup]
	common.ContextObjectNewer[*v1alpha1.DMGroup]
	common.ContextSliceNewer[*v1alpha1.DMGroup, *v1alpha1.DM]

	common.InstanceSliceState[*runtime.DM]
	common.SliceState[*v1alpha1.DM]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.DMGroup]

	stateutil.IFeatureGates
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.DMGroup](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.DMGroup {
	return s.dmg
}

func (s *state) SetObject(dmg *v1alpha1.DMGroup) {
	s.dmg = dmg
}

func (s *state) DMGroup() *v1alpha1.DMGroup {
	return s.dmg
}

func (s *state) Group() *runtime.DMGroup {
	return runtime.FromDMGroup(s.dmg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) DMSlice() []*v1alpha1.DM {
	return s.dms
}

func (s *state) Slice() []*runtime.DM {
	return runtime.FromDMSlice(s.dms)
}

func (s *state) InstanceSlice() []*v1alpha1.DM {
	return s.dms
}

func (s *state) SetInstanceSlice(dms []*v1alpha1.DM) {
	s.dms = dms
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

func (s *state) DMSliceInitializer() common.DMSliceInitializer {
	return common.NewResourceSlice(func(dms []*v1alpha1.DM) { s.dms = dms }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.DMGroup] {
	return common.NewRevision[*runtime.DMGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.dmg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.dmg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.DMGroup](func() *runtime.DMGroup {
			return runtime.FromDMGroup(s.dmg)
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMMaster,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.dmg.Name,
		}
	})
}
