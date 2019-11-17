// +build !ignore_autogenerated

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

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha2

import (
	unsafe "unsafe"

	pingcap "github.com/pingcap/tidb-operator/pkg/apis/pingcap"
	v1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*TidbCluster)(nil), (*pingcap.TidbCluster)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_TidbCluster_To_pingcap_TidbCluster(a.(*TidbCluster), b.(*pingcap.TidbCluster), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*pingcap.TidbCluster)(nil), (*TidbCluster)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_pingcap_TidbCluster_To_v1alpha2_TidbCluster(a.(*pingcap.TidbCluster), b.(*TidbCluster), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*TidbClusterList)(nil), (*pingcap.TidbClusterList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_TidbClusterList_To_pingcap_TidbClusterList(a.(*TidbClusterList), b.(*pingcap.TidbClusterList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*pingcap.TidbClusterList)(nil), (*TidbClusterList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_pingcap_TidbClusterList_To_v1alpha2_TidbClusterList(a.(*pingcap.TidbClusterList), b.(*TidbClusterList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*TidbClusterSpec)(nil), (*pingcap.TidbClusterSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_TidbClusterSpec_To_pingcap_TidbClusterSpec(a.(*TidbClusterSpec), b.(*pingcap.TidbClusterSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*pingcap.TidbClusterSpec)(nil), (*TidbClusterSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_pingcap_TidbClusterSpec_To_v1alpha2_TidbClusterSpec(a.(*pingcap.TidbClusterSpec), b.(*TidbClusterSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*TidbClusterStatus)(nil), (*pingcap.TidbClusterStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_TidbClusterStatus_To_pingcap_TidbClusterStatus(a.(*TidbClusterStatus), b.(*pingcap.TidbClusterStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*pingcap.TidbClusterStatus)(nil), (*TidbClusterStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_pingcap_TidbClusterStatus_To_v1alpha2_TidbClusterStatus(a.(*pingcap.TidbClusterStatus), b.(*TidbClusterStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*pingcap.TidbClusterSpec)(nil), (*TidbClusterSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_pingcap_TidbClusterSpec_To_v1alpha2_TidbClusterSpec(a.(*pingcap.TidbClusterSpec), b.(*TidbClusterSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*pingcap.TidbClusterStatus)(nil), (*TidbClusterStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_pingcap_TidbClusterStatus_To_v1alpha2_TidbClusterStatus(a.(*pingcap.TidbClusterStatus), b.(*TidbClusterStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*TidbClusterSpec)(nil), (*pingcap.TidbClusterSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_TidbClusterSpec_To_pingcap_TidbClusterSpec(a.(*TidbClusterSpec), b.(*pingcap.TidbClusterSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*TidbClusterStatus)(nil), (*pingcap.TidbClusterStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha2_TidbClusterStatus_To_pingcap_TidbClusterStatus(a.(*TidbClusterStatus), b.(*pingcap.TidbClusterStatus), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha2_TidbCluster_To_pingcap_TidbCluster(in *TidbCluster, out *pingcap.TidbCluster, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_TidbClusterSpec_To_pingcap_TidbClusterSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_v1alpha2_TidbClusterStatus_To_pingcap_TidbClusterStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha2_TidbCluster_To_pingcap_TidbCluster is an autogenerated conversion function.
func Convert_v1alpha2_TidbCluster_To_pingcap_TidbCluster(in *TidbCluster, out *pingcap.TidbCluster, s conversion.Scope) error {
	return autoConvert_v1alpha2_TidbCluster_To_pingcap_TidbCluster(in, out, s)
}

func autoConvert_pingcap_TidbCluster_To_v1alpha2_TidbCluster(in *pingcap.TidbCluster, out *TidbCluster, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_pingcap_TidbClusterSpec_To_v1alpha2_TidbClusterSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	if err := Convert_pingcap_TidbClusterStatus_To_v1alpha2_TidbClusterStatus(&in.Status, &out.Status, s); err != nil {
		return err
	}
	return nil
}

// Convert_pingcap_TidbCluster_To_v1alpha2_TidbCluster is an autogenerated conversion function.
func Convert_pingcap_TidbCluster_To_v1alpha2_TidbCluster(in *pingcap.TidbCluster, out *TidbCluster, s conversion.Scope) error {
	return autoConvert_pingcap_TidbCluster_To_v1alpha2_TidbCluster(in, out, s)
}

func autoConvert_v1alpha2_TidbClusterList_To_pingcap_TidbClusterList(in *TidbClusterList, out *pingcap.TidbClusterList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]pingcap.TidbCluster, len(*in))
		for i := range *in {
			if err := Convert_v1alpha2_TidbCluster_To_pingcap_TidbCluster(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1alpha2_TidbClusterList_To_pingcap_TidbClusterList is an autogenerated conversion function.
func Convert_v1alpha2_TidbClusterList_To_pingcap_TidbClusterList(in *TidbClusterList, out *pingcap.TidbClusterList, s conversion.Scope) error {
	return autoConvert_v1alpha2_TidbClusterList_To_pingcap_TidbClusterList(in, out, s)
}

func autoConvert_pingcap_TidbClusterList_To_v1alpha2_TidbClusterList(in *pingcap.TidbClusterList, out *TidbClusterList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TidbCluster, len(*in))
		for i := range *in {
			if err := Convert_pingcap_TidbCluster_To_v1alpha2_TidbCluster(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_pingcap_TidbClusterList_To_v1alpha2_TidbClusterList is an autogenerated conversion function.
func Convert_pingcap_TidbClusterList_To_v1alpha2_TidbClusterList(in *pingcap.TidbClusterList, out *TidbClusterList, s conversion.Scope) error {
	return autoConvert_pingcap_TidbClusterList_To_v1alpha2_TidbClusterList(in, out, s)
}

func autoConvert_v1alpha2_TidbClusterSpec_To_pingcap_TidbClusterSpec(in *TidbClusterSpec, out *pingcap.TidbClusterSpec, s conversion.Scope) error {
	out.Version = in.Version
	if err := Convert_v1alpha2_PDSpec_To_v1alpha1_PDSpec(&in.PD, &out.PD, s); err != nil {
		return err
	}
	if err := Convert_v1alpha2_TiKVSpec_To_v1alpha1_TiKVSpec(&in.TiKV, &out.TiKV, s); err != nil {
		return err
	}
	if err := Convert_v1alpha2_TiDBSpec_To_v1alpha1_TiDBSpec(&in.TiDB, &out.TiDB, s); err != nil {
		return err
	}
	out.PVReclaimPolicy = v1.PersistentVolumeReclaimPolicy(in.PVReclaimPolicy)
	out.PVCDeletePolicy = v1alpha1.PVCDeletePolicy(in.PVCDeletePolicy)
	// WARNING: in.TLSClusterEnabled requires manual conversion: does not exist in peer-type
	out.SchedulerName = in.SchedulerName
	out.Timezone = in.Timezone
	out.ImagePullPolicy = v1.PullPolicy(in.ImagePullPolicy)
	out.HostNetwork = (*bool)(unsafe.Pointer(in.HostNetwork))
	out.Affinity = (*v1.Affinity)(unsafe.Pointer(in.Affinity))
	out.PriorityClassName = in.PriorityClassName
	out.NodeSelector = *(*map[string]string)(unsafe.Pointer(&in.NodeSelector))
	out.Annotations = *(*map[string]string)(unsafe.Pointer(&in.Annotations))
	out.Tolerations = *(*[]v1.Toleration)(unsafe.Pointer(&in.Tolerations))
	return nil
}

func autoConvert_pingcap_TidbClusterSpec_To_v1alpha2_TidbClusterSpec(in *pingcap.TidbClusterSpec, out *TidbClusterSpec, s conversion.Scope) error {
	out.SchedulerName = in.SchedulerName
	if err := Convert_v1alpha1_PDSpec_To_v1alpha2_PDSpec(&in.PD, &out.PD, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_TiDBSpec_To_v1alpha2_TiDBSpec(&in.TiDB, &out.TiDB, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_TiKVSpec_To_v1alpha2_TiKVSpec(&in.TiKV, &out.TiKV, s); err != nil {
		return err
	}
	// WARNING: in.TiKVPromGateway requires manual conversion: does not exist in peer-type
	// WARNING: in.Services requires manual conversion: does not exist in peer-type
	out.PVReclaimPolicy = v1.PersistentVolumeReclaimPolicy(in.PVReclaimPolicy)
	// WARNING: in.EnablePVReclaim requires manual conversion: does not exist in peer-type
	out.Timezone = in.Timezone
	// WARNING: in.EnableTLSCluster requires manual conversion: does not exist in peer-type
	out.Version = in.Version
	out.ImagePullPolicy = v1.PullPolicy(in.ImagePullPolicy)
	out.HostNetwork = (*bool)(unsafe.Pointer(in.HostNetwork))
	out.Affinity = (*v1.Affinity)(unsafe.Pointer(in.Affinity))
	out.PriorityClassName = in.PriorityClassName
	out.NodeSelector = *(*map[string]string)(unsafe.Pointer(&in.NodeSelector))
	out.Annotations = *(*map[string]string)(unsafe.Pointer(&in.Annotations))
	out.Tolerations = *(*[]v1.Toleration)(unsafe.Pointer(&in.Tolerations))
	out.PVCDeletePolicy = PVCDeletePolicy(in.PVCDeletePolicy)
	return nil
}

func autoConvert_v1alpha2_TidbClusterStatus_To_pingcap_TidbClusterStatus(in *TidbClusterStatus, out *pingcap.TidbClusterStatus, s conversion.Scope) error {
	out.ClusterID = in.ClusterID
	if err := Convert_v1alpha2_PDStatus_To_v1alpha1_PDStatus(&in.PD, &out.PD, s); err != nil {
		return err
	}
	if err := Convert_v1alpha2_TiKVStatus_To_v1alpha1_TiKVStatus(&in.TiKV, &out.TiKV, s); err != nil {
		return err
	}
	if err := Convert_v1alpha2_TiDBStatus_To_v1alpha1_TiDBStatus(&in.TiDB, &out.TiDB, s); err != nil {
		return err
	}
	return nil
}

func autoConvert_pingcap_TidbClusterStatus_To_v1alpha2_TidbClusterStatus(in *pingcap.TidbClusterStatus, out *TidbClusterStatus, s conversion.Scope) error {
	out.ClusterID = in.ClusterID
	if err := Convert_v1alpha1_PDStatus_To_v1alpha2_PDStatus(&in.PD, &out.PD, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_TiKVStatus_To_v1alpha2_TiKVStatus(&in.TiKV, &out.TiKV, s); err != nil {
		return err
	}
	if err := Convert_v1alpha1_TiDBStatus_To_v1alpha2_TiDBStatus(&in.TiDB, &out.TiDB, s); err != nil {
		return err
	}
	return nil
}
