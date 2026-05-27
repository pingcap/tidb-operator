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

import (
	"fmt"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func DMGroupInternalServiceName(groupName string) string {
	return fmt.Sprintf("%s-dm-master", groupName)
}

func DMGroupPort(dmg *v1alpha1.DMGroup) int32 {
	if dmg.Spec.Template.Spec.Server.Ports.Port != nil {
		return dmg.Spec.Template.Spec.Server.Ports.Port.Port
	}
	return v1alpha1.DefaultDMPort
}

func DMGroupPeerPort(dmg *v1alpha1.DMGroup) int32 {
	if dmg.Spec.Template.Spec.Server.Ports.PeerPort != nil {
		return dmg.Spec.Template.Spec.Server.Ports.PeerPort.Port
	}
	return v1alpha1.DefaultDMPeerPort
}

func DMPort(dm *v1alpha1.DM) int32 {
	if dm.Spec.Server.Ports.Port != nil {
		return dm.Spec.Server.Ports.Port.Port
	}
	return v1alpha1.DefaultDMPort
}

func DMPeerPort(dm *v1alpha1.DM) int32 {
	if dm.Spec.Server.Ports.PeerPort != nil {
		return dm.Spec.Server.Ports.PeerPort.Port
	}
	return v1alpha1.DefaultDMPeerPort
}

func DMWorkerGroupPort(dwg *v1alpha1.DMWorkerGroup) int32 {
	if dwg.Spec.Template.Spec.Server.Ports.Port != nil {
		return dwg.Spec.Template.Spec.Server.Ports.Port.Port
	}
	return v1alpha1.DefaultDMWorkerPort
}

func DMWorkerPort(dw *v1alpha1.DMWorker) int32 {
	if dw.Spec.Server.Ports.Port != nil {
		return dw.Spec.Server.Ports.Port.Port
	}
	return v1alpha1.DefaultDMWorkerPort
}
