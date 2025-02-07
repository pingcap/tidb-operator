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
	fg      *v1alpha1.TiFlashGroup
	fs      []*v1alpha1.TiFlash

	updateRevision  string
	currentRevision string
	collisionCount  int32
}

type State interface {
	common.TiFlashGroupStateInitializer
	common.ClusterStateInitializer
	common.TiFlashSliceStateInitializer
	common.RevisionStateInitializer[*runtime.TiFlashGroup]

	common.TiFlashGroupState
	common.ClusterState
	common.TiFlashSliceState
	common.RevisionState

	common.GroupState[*runtime.TiFlashGroup]
	common.InstanceSliceState[*runtime.TiFlash]
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) TiFlashGroup() *v1alpha1.TiFlashGroup {
	return s.fg
}

func (s *state) Group() *runtime.TiFlashGroup {
	return runtime.FromTiFlashGroup(s.fg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TiFlashSlice() []*v1alpha1.TiFlash {
	return s.fs
}

func (s *state) Slice() []*runtime.TiFlash {
	return runtime.FromTiFlashSlice(s.fs)
}

func (s *state) TiFlashGroupInitializer() common.TiFlashGroupInitializer {
	return common.NewResource(func(fg *v1alpha1.TiFlashGroup) { s.fg = fg }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) ClusterInitializer() common.ClusterInitializer {
	return common.NewResource(func(cluster *v1alpha1.Cluster) { s.cluster = cluster }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Lazy[string](func() string {
			return s.fg.Spec.Cluster.Name
		})).
		Initializer()
}

func (s *state) TiFlashSliceInitializer() common.TiFlashSliceInitializer {
	return common.NewResourceSlice(func(fs []*v1alpha1.TiFlash) { s.fs = fs }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.TiFlashGroup] {
	return common.NewRevision[*runtime.TiFlashGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.fg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.fg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.TiFlashGroup](func() *runtime.TiFlashGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.fg.Name,
		}
	})
}
