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

package timanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestList(t *testing.T) {
	// test with a corev1.PodList
	podList := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod2",
				},
			},
		},
	}
	list := &List[corev1.Pod, *corev1.Pod]{Items: podList.Items}
	assert.Equal(t, podList.Items, list.Items)
	cp := list.DeepCopyObject()
	assert.Equal(t, list, cp)
}

func TestMap(t *testing.T) {
	m := &Map[string, int]{}

	// Test Store and Load
	m.Store("key1", 1)
	val, ok := m.Load("key1")
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	// Test Load with non-existent key
	val, ok = m.Load("key2")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	// Test Delete
	m.Delete("key1")
	val, ok = m.Load("key1")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	// Test Range
	m.Store("key1", 1)
	m.Store("key2", 2)
	keys := make(map[string]int)
	m.Range(func(k string, v int) bool {
		keys[k] = v
		return true
	})
	assert.Equal(t, map[string]int{"key1": 1, "key2": 2}, keys)

	// Test LoadAndDelete
	m.Store("key3", 3)
	val, ok = m.LoadAndDelete("key3")
	assert.True(t, ok)
	assert.Equal(t, 3, val)
	val, ok = m.Load("key3")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	// Test Swap
	m.Store("key4", 4)
	oldVal, swapped := m.Swap("key4", 5)
	assert.True(t, swapped)
	assert.Equal(t, 4, oldVal)
	val, ok = m.Load("key4")
	assert.True(t, ok)
	assert.Equal(t, 5, val)
}
