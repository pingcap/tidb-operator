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

package podutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsReady(t *testing.T) {
	assert.False(t, IsReady(nil))
	assert.False(t, IsReady(&corev1.Pod{}))
	assert.False(t, IsReady(podWithReadyCondition(corev1.ConditionFalse, metav1.Now())))
	assert.True(t, IsReady(podWithReadyCondition(corev1.ConditionTrue, metav1.Now())))
}

func TestIsAvailable(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name            string
		pod             *corev1.Pod
		minReadySeconds int32
		expected        bool
	}{
		{
			name:     "nil pod",
			expected: false,
		},
		{
			name:            "pod is not ready",
			pod:             podWithReadyCondition(corev1.ConditionFalse, metav1.NewTime(now.Add(-time.Minute))),
			minReadySeconds: 15,
			expected:        false,
		},
		{
			name:            "minimum ready duration is disabled",
			pod:             podWithReadyCondition(corev1.ConditionTrue, metav1.Time{}),
			minReadySeconds: 0,
			expected:        true,
		},
		{
			name:            "pod has not remained ready for the minimum duration",
			pod:             podWithReadyCondition(corev1.ConditionTrue, metav1.NewTime(now.Add(-14*time.Second))),
			minReadySeconds: 15,
			expected:        false,
		},
		{
			name:            "pod has remained ready for the minimum duration",
			pod:             podWithReadyCondition(corev1.ConditionTrue, metav1.NewTime(now.Add(-16*time.Second))),
			minReadySeconds: 15,
			expected:        true,
		},
	}

	for i := range tests {
		test := &tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, IsAvailable(test.pod, test.minReadySeconds, now))
		})
	}
}

func podWithReadyCondition(status corev1.ConditionStatus, transitionTime metav1.Time) *corev1.Pod {
	return &corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:               corev1.PodReady,
					Status:             status,
					LastTransitionTime: transitionTime,
				},
			},
		},
	}
}
