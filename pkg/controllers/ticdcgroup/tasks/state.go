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
	cdcg    *v1alpha1.TiCDCGroup
	cdcs    []*v1alpha1.TiCDC

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool
}

type State interface {
	common.TiCDCSliceStateInitializer
	common.RevisionStateInitializer[*runtime.TiCDCGroup]

	common.TiCDCGroupState
	common.ClusterState
	common.TiCDCSliceState
	common.RevisionState

	common.GroupState[*runtime.TiCDCGroup]

	common.ContextClusterNewer[*v1alpha1.TiCDCGroup]
	common.ContextObjectNewer[*v1alpha1.TiCDCGroup]
	common.ContextSliceNewer[*v1alpha1.TiCDCGroup, *v1alpha1.TiCDC]

	common.InstanceSliceState[*runtime.TiCDC]
	common.SliceState[*v1alpha1.TiCDC]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.TiCDCGroup]
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

func (s *state) Object() *v1alpha1.TiCDCGroup {
	return s.cdcg
}

func (s *state) SetObject(cg *v1alpha1.TiCDCGroup) {
	s.cdcg = cg
}

func (s *state) TiCDCGroup() *v1alpha1.TiCDCGroup {
	return s.cdcg
}

func (s *state) Group() *runtime.TiCDCGroup {
	return runtime.FromTiCDCGroup(s.cdcg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TiCDCSlice() []*v1alpha1.TiCDC {
	return s.cdcs
}

func (s *state) Slice() []*runtime.TiCDC {
	return runtime.FromTiCDCSlice(s.cdcs)
}

func (s *state) InstanceSlice() []*v1alpha1.TiCDC {
	return s.cdcs
}

func (s *state) SetInstanceSlice(cdcs []*v1alpha1.TiCDC) {
	s.cdcs = cdcs
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

func (s *state) TiCDCSliceInitializer() common.TiCDCSliceInitializer {
	return common.NewResourceSlice(func(cdcs []*v1alpha1.TiCDC) { s.cdcs = cdcs }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.TiCDCGroup] {
	return common.NewRevision[*runtime.TiCDCGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.cdcg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.cdcg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.TiCDCGroup](func() *runtime.TiCDCGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiCDC,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.cdcg.Name,
		}
	})
}
