// Copyright 2018 PingCAP, Inc.
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

import (
	"fmt"
	"strings"
)

var latestImageVersion = "latest"

func (dc *DMCluster) Scheme() string {
	if dc.IsTLSClusterEnabled() {
		return "https"
	}
	return "http"
}

func (dc *DMCluster) Timezone() string {
	tz := dc.Spec.Timezone
	if tz == "" {
		return defaultTimeZone
	}
	return tz
}

func (dc *DMCluster) IsTLSClusterEnabled() bool {
	return dc.Spec.TLSCluster != nil && dc.Spec.TLSCluster.Enabled
}

func (dc *DMCluster) MasterAllMembersReady() bool {
	if int(dc.MasterStsDesiredReplicas()) != len(dc.Status.Master.Members) {
		return false
	}

	for _, member := range dc.Status.Master.Members {
		if !member.Health {
			return false
		}
	}
	return true
}

func (dc *DMCluster) WorkerAllMembersReady() bool {
	if int(dc.WorkerStsDesiredReplicas()) != len(dc.Status.Worker.Members) {
		return false
	}

	for _, member := range dc.Status.Worker.Members {
		if member.Stage == "offline" {
			return false
		}
	}
	return true
}

func (dc *DMCluster) MasterStsDesiredReplicas() int32 {
	return dc.Spec.Master.Replicas + int32(len(dc.Status.Master.FailureMembers))
}

func (dc *DMCluster) WorkerStsDesiredReplicas() int32 {
	if dc.Spec.Worker == nil {
		return 0
	}

	return dc.Spec.Worker.Replicas
}

func (dc *DMCluster) GetInstanceName() string {
	return dc.Name
}

func (dc *DMCluster) MasterImage() string {
	image := dc.Spec.Master.Image
	baseImage := dc.Spec.Master.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := dc.Spec.Master.Version
		if version == nil {
			version = &dc.Spec.Version
		}
		if version == nil {
			version = &latestImageVersion
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return image
}

func (dc *DMCluster) WorkerImage() string {
	image := dc.Spec.Worker.Image
	baseImage := dc.Spec.Worker.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := dc.Spec.Worker.Version
		if version == nil {
			version = &dc.Spec.Version
		}
		if version == nil {
			version = &latestImageVersion
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return image
}

func (dc *DMCluster) MasterVersion() string {
	image := dc.MasterImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return latestImageVersion
}

func (dc *DMCluster) MasterUpgrading() bool {
	return dc.Status.Master.Phase == UpgradePhase
}

func (dc *DMCluster) MasterIsAvailable() bool {
	lowerLimit := dc.Spec.Master.Replicas/2 + 1
	if int32(len(dc.Status.Master.Members)) < lowerLimit {
		return false
	}

	var availableNum int32
	for _, masterMember := range dc.Status.Master.Members {
		if masterMember.Health {
			availableNum++
		}
	}

	if availableNum < lowerLimit {
		return false
	}

	if dc.Status.Master.StatefulSet == nil || dc.Status.Master.StatefulSet.ReadyReplicas < lowerLimit {
		return false
	}

	return true
}

func (masterSvc *MasterServiceSpec) GetMasterNodePort() int32 {
	masterNodePortNodePort := masterSvc.MasterNodePort
	if masterNodePortNodePort == nil {
		return 0
	}
	return int32(*masterNodePortNodePort)
}
