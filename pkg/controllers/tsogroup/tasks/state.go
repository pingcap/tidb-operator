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
	tg      *v1alpha1.TSOGroup
	ts      []*v1alpha1.TSO

	pdGroups  []*v1alpha1.PDGroup
	pds       []*v1alpha1.PD
	tsoGroups []*v1alpha1.TSOGroup

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool
	pdmsProtected bool

	stateutil.IFeatureGates
}

type State interface {
	common.ContextObjectNewer[*v1alpha1.TSOGroup]
	common.ContextClusterNewer[*v1alpha1.TSOGroup]
	common.ContextSliceNewer[*v1alpha1.TSOGroup, *v1alpha1.TSO]

	common.RevisionStateInitializer[*runtime.TSOGroup]

	common.ClusterState
	common.InstanceSliceState[*runtime.TSO]
	common.SliceState[*v1alpha1.TSO]
	common.RevisionState

	common.GroupState[*runtime.TSOGroup]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.TSOGroup]

	stateutil.IFeatureGates

	SetPDMSProtected(bool)
	PDMSProtected() bool
	SetPDMSProtectionContext(pdgs []*v1alpha1.PDGroup, pds []*v1alpha1.PD, tgs []*v1alpha1.TSOGroup)
	PDMSProtectionPDGroups() []*v1alpha1.PDGroup
	PDMSProtectionPDs() []*v1alpha1.PD
	PDMSProtectionTSOGroups() []*v1alpha1.TSOGroup
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.TSOGroup](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.TSOGroup {
	return s.tg
}

func (s *state) SetObject(tg *v1alpha1.TSOGroup) {
	s.tg = tg
}

func (s *state) TSOGroup() *v1alpha1.TSOGroup {
	return s.tg
}

func (s *state) Group() *runtime.TSOGroup {
	return runtime.FromTSOGroup(s.tg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TSOSlice() []*v1alpha1.TSO {
	return s.ts
}

func (s *state) PDMSProtectionPDGroups() []*v1alpha1.PDGroup {
	return s.pdGroups
}

func (s *state) PDMSProtectionPDs() []*v1alpha1.PD {
	return s.pds
}

func (s *state) PDMSProtectionTSOGroups() []*v1alpha1.TSOGroup {
	return s.tsoGroups
}

func (s *state) Slice() []*runtime.TSO {
	return runtime.FromTSOSlice(s.ts)
}

func (s *state) InstanceSlice() []*v1alpha1.TSO {
	return s.ts
}

func (s *state) SetInstanceSlice(ts []*v1alpha1.TSO) {
	s.ts = ts
}

func (s *state) SetPDMSProtectionContext(pdgs []*v1alpha1.PDGroup, pds []*v1alpha1.PD, tgs []*v1alpha1.TSOGroup) {
	s.pdGroups = pdgs
	s.pds = pds
	s.tsoGroups = tgs
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

func (s *state) SetPDMSProtected(protected bool) {
	s.pdmsProtected = protected
}

func (s *state) PDMSProtected() bool {
	return s.pdmsProtected
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.TSOGroup] {
	return common.NewRevision[*runtime.TSOGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.tg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.tg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.TSOGroup](func() *runtime.TSOGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTSO,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.tg.Name,
		}
	})
}
