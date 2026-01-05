// Copyright 2026 PingCAP, Inc.
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
	rmg     *v1alpha1.ResourceManagerGroup
	rms     []*v1alpha1.ResourceManager

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool
	stateutil.IFeatureGates
}

type State interface {
	common.ContextObjectNewer[*v1alpha1.ResourceManagerGroup]
	common.ContextClusterNewer[*v1alpha1.ResourceManagerGroup]
	common.ContextSliceNewer[*v1alpha1.ResourceManagerGroup, *v1alpha1.ResourceManager]
	common.ObjectState[*v1alpha1.ResourceManagerGroup]

	common.RevisionStateInitializer[*runtime.ResourceManagerGroup]

	common.ClusterState
	common.InstanceSliceState[*runtime.ResourceManager]
	common.SliceState[*v1alpha1.ResourceManager]
	common.RevisionState

	common.GroupState[*runtime.ResourceManagerGroup]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.ResourceManagerGroup]

	stateutil.IFeatureGates
}

func NewState(key types.NamespacedName) State {
	s := &state{key: key}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.ResourceManagerGroup](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.ResourceManagerGroup {
	return s.rmg
}

func (s *state) SetObject(rmg *v1alpha1.ResourceManagerGroup) {
	s.rmg = rmg
}

func (s *state) ResourceManagerGroup() *v1alpha1.ResourceManagerGroup {
	return s.rmg
}

func (s *state) Group() *runtime.ResourceManagerGroup {
	return runtime.FromResourceManagerGroup(s.rmg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) SetCluster(cluster *v1alpha1.Cluster) {
	s.cluster = cluster
}

func (s *state) ResourceManagerSlice() []*v1alpha1.ResourceManager {
	return s.rms
}

func (s *state) InstanceSlice() []*v1alpha1.ResourceManager {
	return s.rms
}

func (s *state) SetInstanceSlice(rms []*v1alpha1.ResourceManager) {
	s.rms = rms
}

func (s *state) Slice() []*runtime.ResourceManager {
	return runtime.FromResourceManagerSlice(s.rms)
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.ResourceManagerGroup] {
	return common.NewRevision[*runtime.ResourceManagerGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		}),
	).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.rmg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.rmg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.ResourceManagerGroup](func() *runtime.ResourceManagerGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentResourceManager,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.rmg.Name,
		}
	})
}

func (s *state) IsStatusChanged() bool {
	return s.statusChanged
}

func (s *state) SetStatusChanged() {
	s.statusChanged = true
}
