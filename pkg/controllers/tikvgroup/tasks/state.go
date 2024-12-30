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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	kvg     *v1alpha1.TiKVGroup
	kvs     []*v1alpha1.TiKV
}

type State interface {
	common.TiKVGroupStateInitializer
	common.ClusterStateInitializer
	common.TiKVSliceStateInitializer

	common.TiKVGroupState
	common.ClusterState
	common.TiKVSliceState

	common.GroupState[*runtime.TiKVGroup]
	common.InstanceSliceState[*runtime.TiKV]
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) TiKVGroup() *v1alpha1.TiKVGroup {
	return s.kvg
}

func (s *state) Group() *runtime.TiKVGroup {
	return runtime.FromTiKVGroup(s.kvg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TiKVSlice() []*v1alpha1.TiKV {
	return s.kvs
}

func (s *state) Slice() []*runtime.TiKV {
	return runtime.FromTiKVSlice(s.kvs)
}

func (s *state) TiKVGroupInitializer() common.TiKVGroupInitializer {
	return common.NewResource(func(kvg *v1alpha1.TiKVGroup) { s.kvg = kvg }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) ClusterInitializer() common.ClusterInitializer {
	return common.NewResource(func(cluster *v1alpha1.Cluster) { s.cluster = cluster }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.NameFunc(func() string {
			return s.kvg.Spec.Cluster.Name
		})).
		Initializer()
}

func (s *state) TiKVSliceInitializer() common.TiKVSliceInitializer {
	return common.NewResourceSlice(func(kvs []*v1alpha1.TiKV) { s.kvs = kvs }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(common.LabelsFunc(func() map[string]string {
			return map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				v1alpha1.LabelKeyCluster:   s.cluster.Name,
				v1alpha1.LabelKeyGroup:     s.kvg.Name,
			}
		})).
		Initializer()
}
