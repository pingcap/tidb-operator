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

package runtime

import (
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func isAvailable(conds []metav1.Condition, generation, minReadySeconds int64, now time.Time) (bool, time.Duration) {
	cond := meta.FindStatusCondition(conds, v1alpha1.CondReady)
	if cond == nil {
		return false, 0
	}
	if cond.ObservedGeneration != generation {
		return false, 0
	}
	if cond.Status != metav1.ConditionTrue {
		return false, 0
	}

	return isConditionStable(cond, minReadySeconds, now)
}

func isTiKVAvailable(conds []metav1.Condition, generation, minReadySeconds int64, now time.Time) (bool, time.Duration) {
	available, remain := isAvailable(conds, generation, minReadySeconds, now)
	if !available {
		return false, remain
	}

	// TiKV rolling update has one more availability gate than other components:
	// the restarted store must have stopped leader eviction. Apply the same
	// minReadySeconds window to LeadersEvicted=False/NotEvicted so the updater
	// does not move to the next TiKV immediately after the scheduler is removed.
	cond := meta.FindStatusCondition(conds, v1alpha1.TiKVCondLeadersEvicted)
	if cond == nil ||
		cond.ObservedGeneration != generation ||
		cond.Status != metav1.ConditionFalse ||
		cond.Reason != v1alpha1.ReasonNotEvicted {
		return false, 0
	}

	return isConditionStable(cond, minReadySeconds, now)
}

func isConditionStable(cond *metav1.Condition, minReadySeconds int64, now time.Time) (bool, time.Duration) {
	if minReadySeconds <= 0 {
		return true, 0
	}
	if cond.LastTransitionTime.IsZero() {
		return false, 0
	}

	remain := cond.LastTransitionTime.Add(time.Duration(minReadySeconds) * time.Second).Sub(now)
	if remain > 0 {
		return false, remain
	}

	return true, 0
}
