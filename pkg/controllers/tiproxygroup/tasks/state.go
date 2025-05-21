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
	proxyg  *v1alpha1.TiProxyGroup
	proxies []*v1alpha1.TiProxy

	updateRevision  string
	currentRevision string
	collisionCount  int32

	statusChanged bool
}

type State interface {
	common.TiProxySliceStateInitializer
	common.RevisionStateInitializer[*runtime.TiProxyGroup]

	common.TiProxyGroupState
	common.ClusterState
	common.TiProxySliceState
	common.RevisionState

	common.GroupState[*runtime.TiProxyGroup]

	common.ContextClusterNewer[*v1alpha1.TiProxyGroup]
	common.ContextObjectNewer[*v1alpha1.TiProxyGroup]

	common.InstanceSliceState[*runtime.TiProxy]
	common.SliceState[*v1alpha1.TiProxy]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.TiProxyGroup]
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

func (s *state) Object() *v1alpha1.TiProxyGroup {
	return s.proxyg
}

func (s *state) SetObject(proxyg *v1alpha1.TiProxyGroup) {
	s.proxyg = proxyg
}

func (s *state) TiProxyGroup() *v1alpha1.TiProxyGroup {
	return s.proxyg
}

func (s *state) Group() *runtime.TiProxyGroup {
	return runtime.FromTiProxyGroup(s.proxyg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TiProxySlice() []*v1alpha1.TiProxy {
	return s.proxies
}

func (s *state) Slice() []*runtime.TiProxy {
	return runtime.FromTiProxySlice(s.proxies)
}

func (s *state) InstanceSlice() []*v1alpha1.TiProxy {
	return s.proxies
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

func (s *state) TiProxySliceInitializer() common.TiProxySliceInitializer {
	return common.NewResourceSlice(func(proxies []*v1alpha1.TiProxy) { s.proxies = proxies }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(s.Labels()).
		Initializer()
}

func (s *state) RevisionInitializer() common.RevisionInitializer[*runtime.TiProxyGroup] {
	return common.NewRevision[*runtime.TiProxyGroup](
		common.RevisionSetterFunc(func(update, current string, collisionCount int32) {
			s.updateRevision = update
			s.currentRevision = current
			s.collisionCount = collisionCount
		})).
		WithCurrentRevision(common.Lazy[string](func() string {
			return s.proxyg.Status.CurrentRevision
		})).
		WithCollisionCount(common.Lazy[*int32](func() *int32 {
			return s.proxyg.Status.CollisionCount
		})).
		WithParent(common.Lazy[*runtime.TiProxyGroup](func() *runtime.TiProxyGroup {
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
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
			v1alpha1.LabelKeyGroup:     s.proxyg.Name,
		}
	})
}
