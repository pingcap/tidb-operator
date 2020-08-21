// Copyright 2019 PingCAP, Inc.
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

package discovery

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// IsAPIGroupVersionSupported checks if given groupVersion is supported by the cluster.
func IsAPIGroupVersionSupported(discoveryCli discovery.DiscoveryInterface, groupVersion string) (bool, error) {
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return false, err
	}
	apiGroupList, err := discoveryCli.ServerGroups()
	if err != nil {
		return false, err
	}
	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name != gv.Group {
			continue
		}
		for _, version := range apiGroup.Versions {
			if version.GroupVersion == gv.String() {
				return true, nil
			}
		}
	}
	return false, nil
}

// IsAPIGroupSupported checks if given group is supported by the cluster.
func IsAPIGroupSupported(discoveryCli discovery.DiscoveryInterface, group string) (bool, error) {
	apiGroupList, err := discoveryCli.ServerGroups()
	if err != nil {
		return false, err
	}
	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name == group {
			return true, nil
		}
	}
	return false, nil
}
