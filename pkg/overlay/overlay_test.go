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

package overlay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/utils/random"
)

type Case[T any] struct {
	expected T
	dst      T
	src      T
}

type Policy uint

const (
	// NoLimit means all cases will be returned
	NoLimit Policy = 0
	// NoZero means cases contain no zero value
	NoZero Policy = 1 << iota
	// NoNil means cases contain no nil value
	NoNil
	// NoNotEqual means cases only have same src and dst
	NoNotEqual
)

func TestOverlayPod(t *testing.T) {
	base := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"aa": "aa",
				"zz": "123",
			},
			Labels: map[string]string{
				"aa": "aa",
				"zz": "123",
			},
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"aa": "aa",
				"zz": "123",
			},
		},
	}
	overlay := v1alpha1.PodOverlay{
		ObjectMeta: v1alpha1.ObjectMeta{
			Annotations: map[string]string{
				"bb": "bb",
				"zz": "456",
			},
			Labels: map[string]string{
				"bb": "bb",
				"zz": "456",
			},
		},
		Spec: &corev1.PodSpec{
			NodeSelector: map[string]string{
				"bb": "bb",
				"zz": "456",
			},
			TerminationGracePeriodSeconds: ptr.To[int64](100),
		},
	}
	OverlayPod(&base, &overlay)

	assert.Equal(t, map[string]string{
		"aa": "aa",
		"bb": "bb",
		"zz": "456",
	}, base.ObjectMeta.Annotations)
	assert.Equal(t, map[string]string{
		"aa": "aa",
		"bb": "bb",
		"zz": "456",
	}, base.ObjectMeta.Labels)
	assert.Equal(t, map[string]string{
		"aa": "aa",
		"bb": "bb",
		"zz": "123", // special case
	}, base.Spec.NodeSelector)
	assert.Equal(t, int64(100), *base.Spec.TerminationGracePeriodSeconds)
}

func randString() string {
	return random.Random(10)
}

func TestOverlayPodSpec(t *testing.T) {
	cases := constructPodSpec(NoLimit)
	for _, c := range cases {
		overlayPodSpec(&c.dst, &c.src)
		assert.Equal(t, &c.expected, &c.dst)
	}
}

func constructQuantity(_ Policy) []Case[resource.Quantity] {
	return []Case[resource.Quantity]{
		{
			expected: resource.Quantity{},
			dst:      resource.Quantity{},
			src:      resource.Quantity{},
		},
		{
			expected: resource.MustParse("10"),
			dst:      resource.MustParse("20"),
			src:      resource.MustParse("10"),
		},
		{
			expected: resource.MustParse("20"),
			dst:      resource.MustParse("20"),
			src:      resource.MustParse("20"),
		},
	}
}

// TODO: add more cases
func constructObjectMeta(_ Policy) []Case[metav1.ObjectMeta] {
	return []Case[metav1.ObjectMeta]{
		{
			expected: metav1.ObjectMeta{},
			dst:      metav1.ObjectMeta{},
			src:      metav1.ObjectMeta{},
		},
	}
}

func constructMapStringToString(_ Policy) []Case[map[string]string] {
	cases := []Case[map[string]string]{
		{
			expected: nil,
			dst:      nil,
			src:      nil,
		},
		{
			expected: map[string]string{},
			dst:      map[string]string{},
			src:      map[string]string{},
		},
		{
			expected: map[string]string{},
			dst:      map[string]string{},
			src:      nil,
		},
		{
			expected: map[string]string{},
			dst:      nil,
			src:      map[string]string{},
		},
		{
			expected: map[string]string{"aa": "aa"},
			dst:      map[string]string{"aa": "aa"},
			src:      map[string]string{"aa": "aa"},
		},
		{
			expected: map[string]string{"aa": "aa"},
			dst:      map[string]string{"aa": "aa"},
			src:      nil,
		},
		{
			expected: map[string]string{"aa": "aa"},
			dst:      nil,
			src:      map[string]string{"aa": "aa"},
		},
		{
			expected: map[string]string{"aa": "aa", "bb": "bb"},
			dst:      map[string]string{"aa": "aa"},
			src:      map[string]string{"bb": "bb"},
		},
		{
			expected: map[string]string{"aa": "bb"},
			dst:      map[string]string{"aa": "aa"},
			src:      map[string]string{"aa": "bb"},
		},
	}

	return cases
}
