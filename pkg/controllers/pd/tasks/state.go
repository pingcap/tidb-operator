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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	pd      *v1alpha1.PD
	pod     *corev1.Pod
	pds     []*v1alpha1.PD
}

type State interface {
	common.PDStateInitializer
	common.PodStateInitializer
	common.PDSliceStateInitializer

	common.PDState
	common.ClusterState
	common.PodState
	common.PDSliceState

	common.InstanceState[*runtime.PD]

	common.ContextClusterNewer[*v1alpha1.PD]

	SetPod(*corev1.Pod)
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) Object() *v1alpha1.PD {
	return s.pd
}

func (s *state) PD() *v1alpha1.PD {
	return s.pd
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) Pod() *corev1.Pod {
	return s.pod
}

func (s *state) Instance() *runtime.PD {
	return runtime.FromPD(s.pd)
}

func (s *state) SetPod(pod *corev1.Pod) {
	s.pod = pod
}

func (s *state) SetCluster(cluster *v1alpha1.Cluster) {
	s.cluster = cluster
}

func (s *state) PDSlice() []*v1alpha1.PD {
	return s.pds
}

func (s *state) PDInitializer() common.PDInitializer {
	return common.NewResource(func(pd *v1alpha1.PD) { s.pd = pd }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) PodInitializer() common.PodInitializer {
	return common.NewResource(s.SetPod).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Lazy[string](func() string {
			return coreutil.PodName[scope.PD](s.pd)
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
			}
		})).
		Initializer()
}
