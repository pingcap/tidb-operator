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
	dbg     *v1alpha1.TiDBGroup
	dbs     []*v1alpha1.TiDB
}

type State interface {
	common.TiDBGroupStateInitializer
	common.ClusterStateInitializer
	common.TiDBSliceStateInitializer

	common.TiDBGroupState
	common.ClusterState
	common.TiDBSliceState

	common.GroupState[*runtime.TiDBGroup]
	common.InstanceSliceState[*runtime.TiDB]
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) TiDBGroup() *v1alpha1.TiDBGroup {
	return s.dbg
}

func (s *state) Group() *runtime.TiDBGroup {
	return runtime.FromTiDBGroup(s.dbg)
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) TiDBSlice() []*v1alpha1.TiDB {
	return s.dbs
}

func (s *state) Slice() []*runtime.TiDB {
	return runtime.FromTiDBSlice(s.dbs)
}

func (s *state) TiDBGroupInitializer() common.TiDBGroupInitializer {
	return common.NewResource(func(dbg *v1alpha1.TiDBGroup) { s.dbg = dbg }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) ClusterInitializer() common.ClusterInitializer {
	return common.NewResource(func(cluster *v1alpha1.Cluster) { s.cluster = cluster }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.NameFunc(func() string {
			return s.dbg.Spec.Cluster.Name
		})).
		Initializer()
}

func (s *state) TiDBSliceInitializer() common.TiDBSliceInitializer {
	return common.NewResourceSlice(func(dbs []*v1alpha1.TiDB) { s.dbs = dbs }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(common.LabelsFunc(func() map[string]string {
			return map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				v1alpha1.LabelKeyCluster:   s.cluster.Name,
				v1alpha1.LabelKeyGroup:     s.dbg.Name,
			}
		})).
		Initializer()
}
