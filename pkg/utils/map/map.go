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

// Merge merges all maps to a new one.
func Merge[K comparable, V any](maps ...map[K]V) map[K]V {
	return MergeTo(nil, maps...)
}

// MergeTo merges all maps to the original one.
func MergeTo[K comparable, V any](original map[K]V, maps ...map[K]V) map[K]V {
	if original == nil {
		original = make(map[K]V)
	}
	for _, m := range maps {
		for k, v := range m {
			original[k] = v
		}
	}
	return original
}

// Copy returns a copy of the given map.
func Copy[K comparable, V any](originalMap map[K]V) map[K]V {
	if originalMap == nil {
		return nil
	}
	// Create a new map to store the copied key-value pairs with the same capacity as the original map
	copiedMap := make(map[K]V, len(originalMap))

	// Iterate over the original map's key-value pairs
	for key, value := range originalMap {
		// Add the key-value pair into the new map, since the value is not a reference type, it is safe to copy directly
		copiedMap[key] = value
	}

	// Return the deep copied map
	return copiedMap
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
