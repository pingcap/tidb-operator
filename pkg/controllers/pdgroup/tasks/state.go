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
	pdg     *v1alpha1.PDGroup
	pds     []*v1alpha1.PD

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool
}

type State interface {
	common.PDSliceStateInitializer
	common.RevisionStateInitializer[*runtime.PDGroup]

	common.PDGroupState
	common.ClusterState
	common.PDSliceState
	common.RevisionState

	common.GroupState[*runtime.PDGroup]

	common.ContextClusterNewer[*v1alpha1.PDGroup]
	common.ContextObjectNewer[*v1alpha1.PDGroup]
	common.ContextSliceNewer[*v1alpha1.PDGroup, *v1alpha1.PD]

	common.InstanceSliceState[*runtime.PD]
	common.SliceState[*v1alpha1.PD]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.PDGroup]
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

func (s *state) Object() *v1alpha1.PDGroup {
	return s.pdg
}

func (s *state) SetObject(pdg *v1alpha1.PDGroup) {
	s.pdg = pdg
}

func (s *state) PDGroup() *v1alpha1.PDGroup {
	return s.pdg
}

func (s *state) Group() *runtime.PDGroup {
	return runtime.FromPDGroup(s.pdg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) PDSlice() []*v1alpha1.PD {
	return s.pds
}

func (s *state) Slice() []*runtime.PD {
	return runtime.FromPDSlice(s.pds)
}

func (s *state) InstanceSlice() []*v1alpha1.PD {
	return s.pds
}

func (s *state) SetInstanceSlice(pds []*v1alpha1.PD) {
	s.pds = pds
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

func (s *state) PDSliceInitializer() common.PDSliceInitializer {
	return common.NewResourceSlice(func(pds []*v1alpha1.PD) { s.pds = pds }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.PDGroup] {
	return common.NewRevision[*runtime.PDGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.pdg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.pdg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.PDGroup](func() *runtime.PDGroup {
			pdg := s.pdg.DeepCopy()
			// always ignore bootstrapped field in spec
			pdg.Spec.Bootstrapped = false
			return runtime.FromPDGroup(pdg)
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.pdg.Name,
		}
	})
}
