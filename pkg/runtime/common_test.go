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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

const testGeneration int64 = 1

func TestTiKVIsAvailableRequiresLeaderEvictionStopped(t *testing.T) {
	now := time.Unix(1000, 0)

	tests := []struct {
		name           string
		conds          []metav1.Condition
		expected       bool
		expectedRemain time.Duration
	}{
		{
			name: "ready condition is still within min ready seconds",
			conds: []metav1.Condition{
				readyCondition(now.Add(-30 * time.Second)),
				leadersEvictedCondition(metav1.ConditionFalse, v1alpha1.ReasonNotEvicted, now.Add(-2*time.Minute)),
			},
			expectedRemain: 30 * time.Second,
		},
		{
			name: "ready but leader eviction condition missing",
			conds: []metav1.Condition{
				readyCondition(now.Add(-2 * time.Minute)),
			},
		},
		{
			name: "ready but leaders are still evicted",
			conds: []metav1.Condition{
				readyCondition(now.Add(-2 * time.Minute)),
				leadersEvictedCondition(metav1.ConditionTrue, v1alpha1.ReasonEvicted, now.Add(-2*time.Minute)),
			},
		},
		{
			name: "ready but leader eviction just stopped",
			conds: []metav1.Condition{
				readyCondition(now.Add(-2 * time.Minute)),
				leadersEvictedCondition(metav1.ConditionFalse, v1alpha1.ReasonNotEvicted, now.Add(-30*time.Second)),
			},
			expectedRemain: 30 * time.Second,
		},
		{
			name: "ready and leader eviction stopped long enough",
			conds: []metav1.Condition{
				readyCondition(now.Add(-2 * time.Minute)),
				leadersEvictedCondition(metav1.ConditionFalse, v1alpha1.ReasonNotEvicted, now.Add(-61*time.Second)),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tikv := newTiKVWithConditions(tt.conds)
			available, remain := tikv.IsAvailable(60, now)
			assert.Equal(t, tt.expected, available)
			assert.Equal(t, tt.expectedRemain, remain)
		})
	}
}

func TestTiKVIsAvailableWithZeroMinReadySeconds(t *testing.T) {
	now := time.Unix(1000, 0)

	tests := []struct {
		name     string
		conds    []metav1.Condition
		expected bool
	}{
		{
			name: "ready but leader eviction condition missing",
			conds: []metav1.Condition{
				readyCondition(time.Time{}),
			},
		},
		{
			name: "ready but leaders are still evicted",
			conds: []metav1.Condition{
				readyCondition(time.Time{}),
				leadersEvictedCondition(metav1.ConditionTrue, v1alpha1.ReasonEvicted, time.Time{}),
			},
		},
		{
			name: "ready but leader eviction reason is invalid",
			conds: []metav1.Condition{
				readyCondition(time.Time{}),
				leadersEvictedCondition(metav1.ConditionFalse, v1alpha1.ReasonEvicted, time.Time{}),
			},
		},
		{
			name: "ready and leader eviction stopped",
			conds: []metav1.Condition{
				readyCondition(time.Time{}),
				leadersEvictedCondition(metav1.ConditionFalse, v1alpha1.ReasonNotEvicted, time.Time{}),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tikv := newTiKVWithConditions(tt.conds)
			available, remain := tikv.IsAvailable(0, now)
			assert.Equal(t, tt.expected, available)
			assert.Zero(t, remain)
		})
	}
}

func newTiKVWithConditions(conds []metav1.Condition) *TiKV {
	tikv := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "tikv-0",
			Generation: testGeneration,
		},
	}
	tikv.Status.ObservedGeneration = testGeneration
	tikv.Status.Conditions = conds
	return FromTiKV(tikv)
}

func readyCondition(lastTransitionTime time.Time) metav1.Condition {
	return metav1.Condition{
		Type:               v1alpha1.CondReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: testGeneration,
		LastTransitionTime: metav1.NewTime(lastTransitionTime),
	}
}

func leadersEvictedCondition(status metav1.ConditionStatus, reason string, lastTransitionTime time.Time) metav1.Condition {
	return metav1.Condition{
		Type:               v1alpha1.TiKVCondLeadersEvicted,
		Status:             status,
		ObservedGeneration: testGeneration,
		Reason:             reason,
		LastTransitionTime: metav1.NewTime(lastTransitionTime),
	}
}
