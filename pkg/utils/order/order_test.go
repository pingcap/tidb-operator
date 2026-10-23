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

package order

import (
	"cmp"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortPinsSelection(t *testing.T) {
	type instance struct {
		id      int
		version int
	}
	o := New[string](func(a, b instance) int { return cmp.Compare(a.id, b.id) }, func(v instance) int { return v.id })
	_, ok := o.Sort("cluster", nil)
	require.False(t, ok)
	selected, ok := o.Sort("cluster", []instance{{id: 2}, {id: 1}})
	require.True(t, ok)
	require.Equal(t, 1, selected.id)
	selected, ok = o.Sort("cluster", []instance{{id: 0}, {id: 1, version: 2}, {id: 2}})
	require.True(t, ok)
	require.Equal(t, instance{id: 1, version: 2}, selected)
	selected, ok = o.Sort("cluster", []instance{{id: 0}, {id: 2}})
	require.True(t, ok)
	require.Equal(t, 0, selected.id)
	selected, ok = o.Sort("other", []instance{{id: 2}})
	require.True(t, ok)
	require.Equal(t, 2, selected.id)
	_, ok = o.Sort("other", nil)
	require.False(t, ok)
	selected, ok = o.Sort("cluster", []instance{{id: 0}, {id: 2}})
	require.True(t, ok)
	require.Equal(t, 0, selected.id)
}

func TestConcurrentSort(t *testing.T) {
	o := New[string](cmp.Compare[int], func(v int) int { return v })
	var wg sync.WaitGroup
	results := make([]int, 100)
	found := make([]bool, 100)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], found[i] = o.Sort("cluster", []int{2, 1, 0})
		}()
	}
	wg.Wait()
	for i := range results {
		require.True(t, found[i])
		require.Zero(t, results[i])
	}
}
