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

package maputil

import (
	"maps"
	"sync"
)

// Merge merges all maps to a new one.
func Merge[K comparable, V any](ms ...map[K]V) map[K]V {
	return MergeTo(nil, ms...)
}

// MergeTo merges all maps to the original one.
func MergeTo[K comparable, V any](original map[K]V, ms ...map[K]V) map[K]V {
	if original == nil {
		original = make(map[K]V)
	}
	for _, m := range ms {
		maps.Copy(original, m)
	}
	return original
}

// AreEqual checks if two maps are equal.
func AreEqual[K comparable](map1, map2 map[K]string) bool {
	if len(map1) != len(map2) {
		return false
	}
	for k, v1 := range map1 {
		v2, ok := map2[k]
		if !ok || v1 != v2 {
			return false
		}
	}
	return true
}

// Select returns a new map with selected keys and values of the originalMap
func Select[K comparable, V any](originalMap map[K]V, keys ...K) map[K]V {
	ret := make(map[K]V)

	for _, k := range keys {
		v, ok := originalMap[k]
		if ok {
			ret[k] = v
		}
	}

	return ret
}

// Map is a wrapper of sync.Map to avoid type assertion in the outer function
type Map[K comparable, V any] struct {
	sync.Map
}

func (m *Map[K, V]) Load(k K) (_ V, _ bool) {
	val, ok := m.Map.Load(k)
	if !ok {
		return
	}
	return val.(V), true
}

func (m *Map[K, V]) LoadOrStore(k K, v V) (_ V, _ bool) {
	val, ok := m.Map.LoadOrStore(k, v)
	return val.(V), ok
}

func (m *Map[K, V]) Store(k K, v V) {
	m.Map.Store(k, v)
}

func (m *Map[K, V]) Delete(k K) {
	m.Map.Delete(k)
}

func (m *Map[K, V]) Range(f func(K, V) bool) {
	m.Map.Range(func(key, val any) bool {
		k := key.(K)
		v := val.(V)
		return f(k, v)
	})
}

func (m *Map[K, V]) LoadAndDelete(k K) (_ V, _ bool) {
	val, ok := m.Map.LoadAndDelete(k)
	if !ok {
		return
	}
	return val.(V), true
}

func (m *Map[K, V]) Swap(k K, v V) (_ V, _ bool) {
	val, ok := m.Map.Swap(k, v)
	if !ok {
		return
	}
	return val.(V), true
}
