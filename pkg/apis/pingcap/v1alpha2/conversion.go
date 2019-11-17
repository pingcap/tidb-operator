// Copyright 2019. PingCAP, Inc.
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

package v1alpha2

import (
	"unsafe"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
)

// Convert_v1alpha2_TidbClusterSpec_To_pingcap_TidbClusterSpec override the auto-generated conversion
func Convert_v1alpha2_TidbClusterSpec_To_pingcap_TidbClusterSpec(in *TidbClusterSpec, out *pingcap.TidbClusterSpec, s conversion.Scope) error {
	// Plain copy
	out.Version = in.Version
	out.PVReclaimPolicy = v1.PersistentVolumeReclaimPolicy(in.PVReclaimPolicy)
	out.PVCDeletePolicy = v1alpha1.PVCDeletePolicy(in.PVCDeletePolicy)
	out.SchedulerName = in.SchedulerName
	out.Timezone = in.Timezone
	out.ImagePullPolicy = v1.PullPolicy(in.ImagePullPolicy)
	out.HostNetwork = (*bool)(unsafe.Pointer(in.HostNetwork))
	out.Affinity = (*v1.Affinity)(unsafe.Pointer(in.Affinity))
	out.PriorityClassName = in.PriorityClassName
	out.NodeSelector = *(*map[string]string)(unsafe.Pointer(&in.NodeSelector))
	out.Annotations = *(*map[string]string)(unsafe.Pointer(&in.Annotations))
	out.Tolerations = *(*[]v1.Toleration)(unsafe.Pointer(&in.Tolerations))

	// Semantic consistent conversion
	if in.PVCDeletePolicy == DeletePolicyImmediate {
		out.EnablePVReclaim = true
	}
	out.EnableTLSCluster = in.TLSClusterEnabled

	// Convert nested structs
	if err := convert_v1alpha2_PDSpec_To_v1alpha1_PDSpec(&in.PD, &out.PD, in, s); err != nil {
		return err
	}
	if err := convert_v1alpha2_TiKVSpec_To_v1alpha1_TiKVSpec(&in.TiKV, &out.TiKV, s); err != nil {
		return err
	}
	if err := convert_v1alpha2_TiDBSpec_To_v1alpha1_TiDBSpec(&in.TiDB, &out.TiDB, s); err != nil {
		return err
	}

	return nil
}

func convert_v1alpha2_PDSpec_To_v1alpha1_PDSpec(in *PDSpec, out *v1alpha1.PDSpec, inCluster *TidbClusterSpec, s conversion.Scope) error {
	if err := convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_ContainerSpec(&in.ComponentBaseSpec, &out.ContainerSpec, s); err != nil {
		return err
	}
	if err := convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_PodAttributeSpec(&in.ComponentBaseSpec, &out.PodAttributesSpec, s); err != nil {
		return err
	}

	return nil
}

func convert_v1alpha2_TiKVSpec_To_v1alpha1_TiKVSpec(in *TiKVSpec, out *v1alpha1.TiKVSpec, s conversion.Scope) error {
	if err := convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_ContainerSpec(&in.ComponentBaseSpec, &out.ContainerSpec, s); err != nil {
		return err
	}
	if err := convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_PodAttributeSpec(&in.ComponentBaseSpec, &out.PodAttributesSpec, s); err != nil {
		return err
	}

	return nil

}

func convert_v1alpha2_TiDBSpec_To_v1alpha1_TiDBSpec(in *TiDBSpec, out *v1alpha1.TiDBSpec, s conversion.Scope) error {
	if err := convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_ContainerSpec(&in.ComponentBaseSpec, &out.ContainerSpec, s); err != nil {
		return err
	}
	if err := convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_PodAttributeSpec(&in.ComponentBaseSpec, &out.PodAttributesSpec, s); err != nil {
		return err
	}

	return nil
}

func convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_ContainerSpec(in *ComponentBaseSpec, out *v1alpha1.ContainerSpec, s conversion.Scope) error {
	if in.ImagePullPolicy == nil {
		out.ImagePullPolicyIsNil = true
	} else {
		out.ImagePullPolicyIsNil = false
		out.ImagePullPolicy = *in.ImagePullPolicy
	}
	return nil
}

func convert_v1alpha2_ComponentBaseSpec_To_v1alpha1_PodAttributeSpec(in *ComponentBaseSpec, out *v1alpha1.PodAttributesSpec, s conversion.Scope) error {
	out.Version = in.Version
	out.HostNetwork = *(*bool)(unsafe.Pointer(&in.HostNetwork))
	out.Affinity = (*v1.Affinity)(unsafe.Pointer(in.Affinity))
	out.PriorityClassName = in.PriorityClassName
	out.NodeSelector = *(*map[string]string)(unsafe.Pointer(&in.NodeSelector))
	out.Annotations = *(*map[string]string)(unsafe.Pointer(&in.Annotations))
	out.Tolerations = *(*[]v1.Toleration)(unsafe.Pointer(&in.Tolerations))
	return nil
}
//
//func convert_v1alpha2_ResourceRequirement_To_v1alpha1_ResourceRequiremennt(in *ResourceRequirement, out *v1alpha1.ResourceRequirement, inCluster *TidbClusterSpec, s conversion.Scope) {
//
//}

// Convert_pingcap_TidbClusterSpec_To_v1alpha2_TidbClusterSpec override the auto-generated conversion
func Convert_pingcap_TidbClusterSpec_To_v1alpha2_TidbClusterSpec(in *pingcap.TidbClusterSpec, out *TidbClusterSpec, s conversion.Scope) error {
	return nil
}

// Convert_v1alpha2_TidbClusterStatus_To_pingcap_TidbClusterStatus override the auto-generated conversion
func Convert_v1alpha2_TidbClusterStatus_To_pingcap_TidbClusterStatus(in *TidbClusterStatus, out *pingcap.TidbClusterStatus, s conversion.Scope) error {
	return autoConvert_v1alpha2_TidbClusterStatus_To_pingcap_TidbClusterStatus(in, out, s)
}

// Convert_pingcap_TidbClusterStatus_To_v1alpha2_TidbClusterStatus override the auto-generated conversion
func Convert_pingcap_TidbClusterStatus_To_v1alpha2_TidbClusterStatus(in *pingcap.TidbClusterStatus, out *TidbClusterStatus, s conversion.Scope) error {
	return autoConvert_pingcap_TidbClusterStatus_To_v1alpha2_TidbClusterStatus(in, out, s)
}
