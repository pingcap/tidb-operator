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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestCacheTTLRemaining(t *testing.T) {
	now := time.Unix(100, 0)

	tests := []struct {
		name     string
		tikv     *v1alpha1.TiKV
		expected time.Duration
	}{
		{
			name: "ttl is unset",
			tikv: &v1alpha1.TiKV{},
		},
		{
			name: "leaders are not evicted",
			tikv: &v1alpha1.TiKV{
				Spec: v1alpha1.TiKVSpec{CacheTTLSeconds: ptr.To[int64](30)},
			},
		},
		{
			name: "ttl has not expired",
			tikv: &v1alpha1.TiKV{
				Spec: v1alpha1.TiKVSpec{CacheTTLSeconds: ptr.To[int64](30)},
				Status: v1alpha1.TiKVStatus{CommonStatus: v1alpha1.CommonStatus{
					Conditions: []metav1.Condition{{
						Type:               v1alpha1.TiKVCondLeadersEvicted,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(now.Add(-10 * time.Second)),
					}},
				}},
			},
			expected: 20 * time.Second,
		},
		{
			name: "ttl has expired",
			tikv: &v1alpha1.TiKV{
				Spec: v1alpha1.TiKVSpec{CacheTTLSeconds: ptr.To[int64](30)},
				Status: v1alpha1.TiKVStatus{CommonStatus: v1alpha1.CommonStatus{
					Conditions: []metav1.Condition{{
						Type:               v1alpha1.TiKVCondLeadersEvicted,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(now.Add(-31 * time.Second)),
					}},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, cacheTTLRemaining(tt.tikv, now))
		})
	}
}
