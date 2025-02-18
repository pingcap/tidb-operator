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

package util

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

// NOTE: copy from tidb-operator/pkg/util/

// TODO(ideascf): copy UT
// AppendOverwriteEnv appends envs b into a and overwrites the envs whose names already exist
// in b.
// Note that this will not change relative order of envs.
func AppendOverwriteEnv(a []corev1.EnvVar, b []corev1.EnvVar) []corev1.EnvVar {
	for _, valNew := range b {
		matched := false
		for j, valOld := range a {
			// It's possible there are multiple instances of the same variable in this array,
			// so we just overwrite all of them rather than trying to resolve dupes here.
			if valNew.Name == valOld.Name {
				a[j] = valNew
				matched = true
			}
		}
		if !matched {
			a = append(a, valNew)
		}
	}
	return a
}

// TODO(ideascf): copy UT
// AppendEnvIfPresent appends the given environment if present
func AppendEnvIfPresent(envs []corev1.EnvVar, name string) []corev1.EnvVar {
	for _, e := range envs {
		if e.Name == name {
			return envs
		}
	}
	if val, ok := os.LookupEnv(name); ok {
		envs = append(envs, corev1.EnvVar{
			Name:  name,
			Value: val,
		})
	}
	return envs
}

// TODO(ideascf): copy UT
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

// TODO(ideascf): copy UT
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
