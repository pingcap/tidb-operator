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
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	pdg     *v1alpha1.PDGroup
	pds     []*v1alpha1.PD
}

type State interface {
	common.PDGroupStateInitializer
	common.ClusterStateInitializer
	common.PDSliceStateInitializer

	common.PDGroupState
	common.ClusterState
	common.PDSliceState
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) PDGroup() *v1alpha1.PDGroup {
	return s.pdg
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) PDSlice() []*v1alpha1.PD {
	return s.pds
}

func (s *state) PDGroupInitializer() common.PDGroupInitializer {
	return common.NewResource(func(pdg *v1alpha1.PDGroup) { s.pdg = pdg }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) ClusterInitializer() common.ClusterInitializer {
	return common.NewResource(func(cluster *v1alpha1.Cluster) { s.cluster = cluster }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.NameFunc(func() string {
			return s.pdg.Spec.Cluster.Name
		})).
		Initializer()
}

func (s *state) PDSliceInitializer() common.PDSliceInitializer {
	return common.NewResourceSlice(func(pds []*v1alpha1.PD) { s.pds = pds }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithLabels(common.LabelsFunc(func() map[string]string {
			return map[string]string{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   s.cluster.Name,
				v1alpha1.LabelKeyGroup:     s.pdg.Name,
			}
		})).
		Initializer()
}
