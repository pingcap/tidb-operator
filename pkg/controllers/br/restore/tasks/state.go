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

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type (
	RestoreInitializer = common.ResourceInitializer[brv1alpha1.Restore]

	RestoreStateInitializer interface {
		RestoreInitializer() RestoreInitializer
	}
	RestoreState interface {
		Restore() *brv1alpha1.Restore
	}
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	restore *brv1alpha1.Restore
}

type State interface {
	RestoreStateInitializer
	RestoreState
	common.ClusterState
	common.ContextClusterNewer[*brv1alpha1.Restore]
	common.JobState[*runtime.Restore]
}

var _ State = &state{}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) Restore() *brv1alpha1.Restore {
	return s.restore
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) SetCluster(cluster *v1alpha1.Cluster) {
	s.cluster = cluster
}

func (s *state) Object() *brv1alpha1.Restore {
	return s.restore
}

func (s *state) RestoreInitializer() RestoreInitializer {
	return common.NewResource(func(restore *brv1alpha1.Restore) { s.restore = restore }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) Labels() common.LabelsOption {
	return common.Lazy[map[string]string](func() map[string]string {
		return map[string]string{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyComponent: brv1alpha1.LabelValComponentRestore,
			v1alpha1.LabelKeyCluster:   s.cluster.Name,
		}
	})
}

func (s *state) Job() *runtime.Restore {
	return (*runtime.Restore)(s.restore)
}
