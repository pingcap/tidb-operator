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
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	tiproxy *v1alpha1.TiProxy
	pod     *corev1.Pod

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	isPodTerminating bool

	statusChanged bool

	healthy bool

	stateutil.IFeatureGates
	stateutil.IPDClient
}

type State interface {
	common.TiProxyState
	common.ClusterState

	common.PodState
	common.PodStateUpdater

	common.InstanceState[*runtime.TiProxy]

	common.ContextClusterNewer[*v1alpha1.TiProxy]
	common.ContextObjectNewer[*v1alpha1.TiProxy]

	common.StatusPersister[*v1alpha1.TiProxy]
	common.StatusUpdater

	common.HealthyState
	common.HealthyStateUpdater

	stateutil.IFeatureGates
	stateutil.IPDClient
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.TiProxy](s)
	s.IPDClient = stateutil.NewPDClientState(s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.TiProxy {
	return s.tiproxy
}

func (s *state) SetObject(tiproxy *v1alpha1.TiProxy) {
	s.tiproxy = tiproxy
}

func (s *state) TiProxy() *v1alpha1.TiProxy {
	return s.tiproxy
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

func (s *state) Instance() *runtime.TiProxy {
	return runtime.FromTiProxy(s.tiproxy)
}

func (s *state) IsStatusChanged() bool {
	return s.statusChanged
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

func (s *state) SetStatusChanged() {
	s.statusChanged = true
}

func (s *state) IsHealthy() bool {
	return s.healthy
}

func (s *state) SetHealthy() {
	s.healthy = true
}

func (s *state) GetServerLabels() map[string]string {
	return s.tiproxy.Spec.Server.Labels
}
