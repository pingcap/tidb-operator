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

package v1alpha1

import "fmt"

func (tngm *TidbNGMonitoring) GetInstanceName() string {
	return tngm.Name
}

// NGMonitoringImage return the image used by NGMonitoring.
func (tngm *TidbNGMonitoring) NGMonitoringImage() string {
	image := tngm.Spec.NGMonitoring.Image
	baseImage := tngm.Spec.NGMonitoring.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tngm.Spec.NGMonitoring.Version
		if version == nil {
			version = tngm.Spec.Version
		}
		if *version == "" {
			image = baseImage
		} else {
			image = fmt.Sprintf("%s:%s", baseImage, *version)
		}
	}
	return image
}
