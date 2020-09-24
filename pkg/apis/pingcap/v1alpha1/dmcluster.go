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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/util/sets"
)

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

func (dc *DMCluster) IsPVReclaimEnabled() bool {
	enabled := dc.Spec.EnablePVReclaim
	if enabled == nil {
		return defaultEnablePVReclaim
	}
	return *enabled
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

func (dc *DMCluster) MasterAutoFailovering() bool {
	if len(dc.Status.Master.FailureMembers) == 0 {
		return false
	}

	for _, failureMember := range dc.Status.Master.FailureMembers {
		if !failureMember.MemberDeleted {
			return true
		}
	}
	return false
}

func (dc *DMCluster) MasterStsDesiredReplicas() int32 {
	return dc.Spec.Master.Replicas + int32(len(dc.Status.Master.FailureMembers))
}

func (dc *DMCluster) MasterStsActualReplicas() int32 {
	stsStatus := dc.Status.Master.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (dc *DMCluster) MasterStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	replicas := dc.Spec.Master.Replicas
	if !excludeFailover {
		replicas = dc.MasterStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, dc.getDeleteSlots(label.DMMasterLabelVal))
}

func (dc *DMCluster) WorkerStsActualReplicas() int32 {
	stsStatus := dc.Status.Worker.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (dc *DMCluster) WorkerStsDesiredReplicas() int32 {
	if dc.Spec.Worker == nil {
		return 0
	}

	return dc.Spec.Worker.Replicas + int32(len(dc.Status.Worker.FailureMembers))
}

func (dc *DMCluster) WorkerStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	if dc.Spec.Worker == nil {
		return sets.Int32{}
	}
	replicas := dc.Spec.Worker.Replicas
	if !excludeFailover {
		replicas = dc.WorkerStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, dc.getDeleteSlots(label.DMWorkerLabelVal))
}

func (dc *DMCluster) GetInstanceName() string {
	return dc.Name
}

func (dc *DMCluster) MasterImage() string {
	image := dc.Spec.Master.BaseImage
	version := dc.Spec.Master.Version
	if version == nil {
		version = &dc.Spec.Version
	}
	if *version != "" {
		image = fmt.Sprintf("%s:%s", image, *version)
	}
	return image
}

func (dc *DMCluster) WorkerImage() string {
	image := dc.Spec.Worker.BaseImage
	version := dc.Spec.Worker.Version
	if version == nil {
		version = &dc.Spec.Version
	}
	if *version != "" {
		image = fmt.Sprintf("%s:%s", image, *version)
	}
	return image
}

func (dc *DMCluster) MasterVersion() string {
	image := dc.MasterImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return "latest"
}

func (dc *DMCluster) MasterUpgrading() bool {
	return dc.Status.Master.Phase == UpgradePhase
}

func (dc *DMCluster) MasterScaling() bool {
	return dc.Status.Master.Phase == ScalePhase
}

func (dc *DMCluster) getDeleteSlots(component string) (deleteSlots sets.Int32) {
	deleteSlots = sets.NewInt32()
	annotations := dc.GetAnnotations()
	if annotations == nil {
		return deleteSlots
	}
	var key string
	if component == label.DMMasterLabelVal {
		key = label.AnnDMMasterDeleteSlots
	} else if component == label.DMWorkerLabelVal {
		key = label.AnnDMWorkerDeleteSlots
	} else {
		return
	}
	value, ok := annotations[key]
	if !ok {
		return
	}
	var slice []int32
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deleteSlots.Insert(slice...)
	return
}

func (dc *DMCluster) MasterAllPodsStarted() bool {
	return dc.MasterStsDesiredReplicas() == dc.MasterStsActualReplicas()
}

func (dc *DMCluster) WorkerAllPodsStarted() bool {
	return dc.WorkerStsDesiredReplicas() == dc.WorkerStsActualReplicas()
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
