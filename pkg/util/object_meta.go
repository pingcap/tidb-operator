// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package util

// Utilities here are used for modifying the ObjectMeta field for k8s object

// CombineStringMap merges maps into a new map.
// NOTE: if the same key exists in multiple source maps, the value of the first one will be kept.
// so we suggest to :
//   - pass the map generated by TiDB-Operator as the first argument.
//   - pass components' map (labels or annotations) before cluster's.
func CombineStringMap(maps ...map[string]string) map[string]string {
	r := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			if _, ok := r[k]; !ok {
				r[k] = v
			}
		}
	}
	return r
}

// CopyStringMap copy annotations to a new string map
func CopyStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := map[string]string{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
