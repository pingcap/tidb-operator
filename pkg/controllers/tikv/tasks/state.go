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
	tikv    *v1alpha1.TiKV
	pod     *corev1.Pod
}

type State interface {
	common.TiKVStateInitializer
	common.PodStateInitializer

	common.TiKVState
	common.ClusterState
	common.PodState

	common.InstanceState[*runtime.TiKV]

	common.ContextClusterNewer[*v1alpha1.TiKV]

	SetPod(*corev1.Pod)
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) Object() *v1alpha1.TiKV {
	return s.tikv
}

func (s *state) TiKV() *v1alpha1.TiKV {
	return s.tikv
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) Pod() *corev1.Pod {
	return s.pod
}

func (s *state) Instance() *runtime.TiKV {
	return runtime.FromTiKV(s.tikv)
}

func (s *state) SetPod(pod *corev1.Pod) {
	s.pod = pod
}

func (s *state) SetCluster(cluster *v1alpha1.Cluster) {
	s.cluster = cluster
}

func (s *state) TiKVInitializer() common.TiKVInitializer {
	return common.NewResource(func(tikv *v1alpha1.TiKV) { s.tikv = tikv }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) PodInitializer() common.PodInitializer {
	return common.NewResource(s.SetPod).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Lazy[string](func() string {
			return coreutil.PodName[scope.TiKV](s.tikv)
		})).
		Initializer()
}
