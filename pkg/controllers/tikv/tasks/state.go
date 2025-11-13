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
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
)

const (
	// minRegionCountForLeaderCountCheck is the minimum region count for leader count check.
	// If the region count is less than this value, we will not check the leader count.
	minRegionCountForLeaderCountCheck = 100

	// Now we have to wait an arbitrary duration to ensure that is_busy has been reported to the PD
	minReadySeconds       = 15
	minReadySecondsJitter = 3
)

type state struct {
	key types.NamespacedName

	cluster *v1alpha1.Cluster
	tikv    *v1alpha1.TiKV
	pod     *corev1.Pod

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	isPodTerminating bool

	storeState    pdv1.NodeState
	statusChanged bool
	leaderCount   int
	regionCount   int
	storeBusy     bool

	healthy bool

	stateutil.IFeatureGates
}

type State interface {
	common.TiKVState
	common.ClusterState

	common.PodState
	common.PodStateUpdater

	common.InstanceState[*runtime.TiKV]

	common.ContextClusterNewer[*v1alpha1.TiKV]
	common.ContextObjectNewer[*v1alpha1.TiKV]

	common.StatusUpdater
	common.StatusPersister[*v1alpha1.TiKV]

	common.StoreState
	common.StoreStateUpdater

	common.HealthyState
	common.HealthyStateUpdater

	stateutil.IFeatureGates
}

func NewState(key types.NamespacedName) State {
	s := &state{
		key: key,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.TiKV](s)
	return s
}

func (s *state) Key() types.NamespacedName {
	return s.key
}

func (s *state) Object() *v1alpha1.TiKV {
	return s.tikv
}

func (s *state) SetObject(tikv *v1alpha1.TiKV) {
	s.tikv = tikv
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

func (s *state) GetStoreState() pdv1.NodeState {
	return s.storeState
}

func (s *state) GetLeaderCount() int {
	return s.leaderCount
}

func (s *state) GetRegionCount() int {
	return s.regionCount
}

func (s *state) SetStoreState(state pdv1.NodeState) {
	s.storeState = state
}

func (s *state) SetLeaderCount(count int) {
	s.leaderCount = count
}

func (s *state) SetRegionCount(count int) {
	s.regionCount = count
}

func (s *state) IsStoreUp() bool {
	return s.storeState == v1alpha1.StoreStatePreparing || s.storeState == v1alpha1.StoreStateServing
}

func (s *state) IsHealthy() bool {
	return s.healthy
}

func (s *state) SetHealthy() {
	// We should also check if the leader count is enough.
	// Especially when rolling restart tikv, we need to ensure the leader count of restarted tikv is enough.
	s.healthy = true
}

func (s *state) IsStoreBusy() bool {
	return s.storeBusy
}

func (s *state) SetStoreBusy(busy bool) {
	s.storeBusy = busy
}

// isLeaderCountEnough checks if the leader count is enough.
// If the region count is greater than 100, we will check if the leader count is greater than 0.
// Otherwise, we will return true because too small total region count.
func isLeaderCountEnough(leaderCount, regionCount int) bool {
	if regionCount >= minRegionCountForLeaderCountCheck {
		return leaderCount > 0
	}
	return true
}

func IsStoreReady(s State) bool {
	return !s.IsStoreBusy() && isLeaderCountEnough(s.GetLeaderCount(), s.GetRegionCount())
}
