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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	tiflash *v1alpha1.TiFlash
	pod     *corev1.Pod
}

type State interface {
	common.TiFlashStateInitializer
	common.ClusterStateInitializer
	common.PodStateInitializer

	common.TiFlashState
	common.ClusterState
	common.PodState

	common.InstanceState[*runtime.TiFlash]

	SetPod(*corev1.Pod)
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	return s
}

func (s *state) TiFlash() *v1alpha1.TiFlash {
	return s.tiflash
}

func (s *state) Cluster() *v1alpha1.Cluster {
	return s.cluster
}

func (s *state) Pod() *corev1.Pod {
	return s.pod
}

func (s *state) Instance() *runtime.TiFlash {
	return runtime.FromTiFlash(s.tiflash)
}

func (s *state) SetPod(pod *corev1.Pod) {
	s.pod = pod
}

func (s *state) TiFlashInitializer() common.TiFlashInitializer {
	return common.NewResource(func(tiflash *v1alpha1.TiFlash) { s.tiflash = tiflash }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) ClusterInitializer() common.ClusterInitializer {
	return common.NewResource(func(cluster *v1alpha1.Cluster) { s.cluster = cluster }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Lazy[string](func() string {
			return s.tiflash.Spec.Cluster.Name
		})).
		Initializer()
}

func (s *state) PodInitializer() common.PodInitializer {
	return common.NewResource(s.SetPod).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Lazy[string](func() string {
			return s.tiflash.PodName()
		})).
		Initializer()
}
