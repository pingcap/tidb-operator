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
	"slices"
	"sync"
)

// Order keeps the selected key while it remains in the supplied values.
type Order[G, K comparable, T any] interface {
	Sort(G, []T) (T, bool)
}

type order[G, K comparable, T any] struct {
	mu       sync.Mutex
	selected map[G]K
	compare  func(T, T) int
	key      func(T) K
}

func New[G, K comparable, T any](compare func(T, T) int, key func(T) K) Order[G, K, T] {
	return &order[G, K, T]{selected: make(map[G]K), compare: compare, key: key}
}

// Sort returns the selected value from this call's input, choosing a new top
// only when the previous selection is absent. In that case it sorts values in place.
// It does not retain the values.
func (o *order[G, K, T]) Sort(id G, values []T) (T, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(values) == 0 {
		delete(o.selected, id)
		var zero T
		return zero, false
	}
	if key, ok := o.selected[id]; ok {
		for _, value := range values {
			if o.key(value) == key {
				return value, true
			}
		}
	}
	slices.SortFunc(values, o.compare)
	top := values[0]
	o.selected[id] = o.key(top)
	return top, true
}
