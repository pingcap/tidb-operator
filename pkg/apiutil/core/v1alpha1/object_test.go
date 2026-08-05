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

package coreutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func TestIsStatusConditionTrueForCluster(t *testing.T) {
	const (
		generation        = int64(7)
		clusterGeneration = int64(11)
	)

	newGroup := func() *v1alpha1.PDGroup {
		return &v1alpha1.PDGroup{
			ObjectMeta: metav1.ObjectMeta{Generation: generation},
			Status: v1alpha1.PDGroupStatus{CommonStatus: v1alpha1.CommonStatus{
				ObservedGeneration:        generation,
				ObservedClusterGeneration: clusterGeneration,
				Conditions: []metav1.Condition{{
					Type:               v1alpha1.CondSuspended,
					Status:             metav1.ConditionTrue,
					ObservedGeneration: generation,
				}},
			}},
		}
	}

	tests := []struct {
		name   string
		mutate func(*v1alpha1.PDGroup)
		want   bool
	}{
		{name: "all generations current", want: true},
		{
			name: "status generation stale",
			mutate: func(group *v1alpha1.PDGroup) {
				group.Status.ObservedGeneration--
			},
		},
		{
			name: "cluster generation stale",
			mutate: func(group *v1alpha1.PDGroup) {
				group.Status.ObservedClusterGeneration--
			},
		},
		{
			name: "condition generation stale",
			mutate: func(group *v1alpha1.PDGroup) {
				group.Status.Conditions[0].ObservedGeneration--
			},
		},
		{
			name: "condition false",
			mutate: func(group *v1alpha1.PDGroup) {
				group.Status.Conditions[0].Status = metav1.ConditionFalse
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := newGroup()
			if tt.mutate != nil {
				tt.mutate(group)
			}
			assert.Equal(t, tt.want, IsStatusConditionTrueForCluster[scope.PDGroup](
				group,
				v1alpha1.CondSuspended,
				clusterGeneration,
			))
		})
	}
}
