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

import "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"

func hostToURL(host string, isTLS bool) string {
	scheme := "http"
	if isTLS {
		scheme = "https"
	}

	return scheme + "://" + host
}

var allMainContainers = map[string]struct{}{
	v1alpha1.ContainerNamePD:        {},
	v1alpha1.ContainerNameTiKV:      {},
	v1alpha1.ContainerNameTiDB:      {},
	v1alpha1.ContainerNameTiFlash:   {},
	v1alpha1.ContainerNameTiCDC:     {},
	v1alpha1.ContainerNameTSO:       {},
	v1alpha1.ContainerNameScheduler: {},
	v1alpha1.ContainerNameTiProxy:   {},
}

// IsMainContainer checks whether the container is a main container
// Main container means the main component container of an instance
func IsMainContainer(name string) bool {
	if _, ok := allMainContainers[name]; ok {
		return true
	}

	return false
}

func persistentVolumeClaimName(podName, volName string) string {
	// ref: https://github.com/pingcap/tidb-operator/blob/v1.6.0/pkg/apis/pingcap/v1alpha1/helpers.go#L92
	// NOTE: for v1, should use component as volName of data, e.g. pd
	return volName + "-" + podName
}
