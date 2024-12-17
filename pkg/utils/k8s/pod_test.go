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

package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestComparePods(t *testing.T) {
	tests := []struct {
		name     string
		current  *corev1.Pod
		expected *corev1.Pod
		want     CompareResult
	}{
		{
			name: "test equal",
			current: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
			),
			expected: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
			),
			want: CompareResultEqual,
		},
		{
			name: "revision should not be ignored",
			current: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
				fake.Label[corev1.Pod](v1alpha1.LabelKeyInstanceRevisionHash, "v2"),
			),
			expected: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
				fake.Label[corev1.Pod](v1alpha1.LabelKeyInstanceRevisionHash, "v1"),
			),
			want: CompareResultUpdate,
		},
		{
			name: "only labels different",
			current: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
				fake.Label[corev1.Pod]("test", "bar"),
				fake.Label[corev1.Pod](v1alpha1.LabelKeyInstanceRevisionHash, "v2"),
			),
			expected: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
				fake.Label[corev1.Pod]("test", "test"),
				fake.Label[corev1.Pod](v1alpha1.LabelKeyInstanceRevisionHash, "v1"),
			),
			want: CompareResultUpdate,
		},
		{
			name: "only annotations different",
			current: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
				fake.Label[corev1.Pod]("test", "bar"),
				fake.Annotation[corev1.Pod]("k1", "v1"),
			),
			expected: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
				fake.Label[corev1.Pod]("test", "bar"),
				fake.Annotation[corev1.Pod]("k1", "v2"),
			),
			want: CompareResultUpdate,
		},
		{
			name: "test recreate",
			current: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "foo"),
			),
			expected: fake.FakeObj("pod",
				fake.Label[corev1.Pod](v1alpha1.LabelKeyPodSpecHash, "bar"),
			),
			want: CompareResultRecreate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComparePods(tt.current, tt.expected); got != tt.want {
				t.Errorf("ComparePods() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateHashAndSetLabels(t *testing.T) {
	p1 := fake.FakeObj("pod", func(p *corev1.Pod) *corev1.Pod {
		p.Spec.Containers = []corev1.Container{
			{Name: "test", Image: "test"},
		}
		p.Spec.TerminationGracePeriodSeconds = ptr.To(int64(10))
		return p
	})

	p2 := p1.DeepCopy()
	if p2.Labels == nil {
		p2.Labels = map[string]string{}
	}
	p2.Labels["foo"] = "bar"

	CalculateHashAndSetLabels(p1)
	CalculateHashAndSetLabels(p2)
	if p1.Labels[v1alpha1.LabelKeyPodSpecHash] != p2.Labels[v1alpha1.LabelKeyPodSpecHash] {
		t.Errorf("CalculateHashAndSetLabels() = %v, want %v", p1.Labels[v1alpha1.LabelKeyPodSpecHash], p2.Labels[v1alpha1.LabelKeyPodSpecHash])
	}
}
