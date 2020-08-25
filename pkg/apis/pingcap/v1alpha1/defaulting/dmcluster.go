// Copyright 2020 PingCAP, Inc.
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

package defaulting

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	defaultMasterImage = "pingcap/dm"
	defaultWorkerImage = "pingcap/dm"
)

func SetDMClusterDefault(dc *v1alpha1.DMCluster) {
	setDMClusterSpecDefault(dc)
	setMasterSpecDefault(dc)
	if dc.Spec.Worker != nil {
		setWorkerSpecDefault(dc)
	}
}

// setDMClusterSpecDefault is only managed the property under Spec
func setDMClusterSpecDefault(dc *v1alpha1.DMCluster) {
	if string(dc.Spec.ImagePullPolicy) == "" {
		dc.Spec.ImagePullPolicy = corev1.PullIfNotPresent
	}
	if dc.Spec.TLSCluster == nil {
		dc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: false}
	}
	if dc.Spec.EnablePVReclaim == nil {
		d := false
		dc.Spec.EnablePVReclaim = &d
	}
	retainPVP := corev1.PersistentVolumeReclaimRetain
	if dc.Spec.PVReclaimPolicy == nil {
		dc.Spec.PVReclaimPolicy = &retainPVP
	}
}

func setMasterSpecDefault(dc *v1alpha1.DMCluster) {
	if len(dc.Spec.Version) > 0 || dc.Spec.Master.Version != nil {
		if dc.Spec.Master.BaseImage == "" {
			dc.Spec.Master.BaseImage = defaultMasterImage
		}
	}
	if dc.Spec.Master.MaxFailoverCount == nil {
		dc.Spec.Master.MaxFailoverCount = pointer.Int32Ptr(3)
	}
}

func setWorkerSpecDefault(dc *v1alpha1.DMCluster) {
	if len(dc.Spec.Version) > 0 || dc.Spec.Worker.Version != nil {
		if dc.Spec.Worker.BaseImage == "" {
			dc.Spec.Worker.BaseImage = defaultWorkerImage
		}
	}
	if dc.Spec.Worker.MaxFailoverCount == nil {
		dc.Spec.Worker.MaxFailoverCount = pointer.Int32Ptr(3)
	}
}
