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
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	tikv    *v1alpha1.TiKV
	pod     *corev1.Pod

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	isPodTerminating bool

	storeState    string
	statusChanged bool
}

type State interface {
	common.TiKVStateInitializer

	common.TiKVState
	common.ClusterState

	common.PodState
	common.PodStateUpdater

	common.InstanceState[*runtime.TiKV]

	common.ContextClusterNewer[*v1alpha1.TiKV]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.TiKV]

	common.StoreState
	common.StoreStateUpdater

	common.HealthyState
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

func (s *state) IsPodTerminating() bool {
	return s.isPodTerminating
}

func (s *state) Instance() *runtime.TiKV {
	return runtime.FromTiKV(s.tikv)
}

func (s *state) SetPod(pod *corev1.Pod) {
	s.pod = pod
	if pod != nil && !pod.GetDeletionTimestamp().IsZero() {
		s.isPodTerminating = true
	}
}

func (s *state) DeletePod(pod *corev1.Pod) {
	s.isPodTerminating = true
	s.pod = pod
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

func (s *state) TiKVInitializer() common.TiKVInitializer {
	return common.NewResource(func(tikv *v1alpha1.TiKV) { s.tikv = tikv }).
		WithNamespace(common.Namespace(s.key.Namespace)).
		WithName(common.Name(s.key.Name)).
		Initializer()
}

func (s *state) GetStoreState() string {
	return s.storeState
}

func (s *state) SetStoreState(state string) {
	s.storeState = state
}

func (s *state) IsStoreUp() bool {
	return s.storeState == v1alpha1.StoreStatePreparing || s.storeState == v1alpha1.StoreStateServing
}

func (s *state) IsHealthy() bool {
	return s.IsStoreUp()
}
