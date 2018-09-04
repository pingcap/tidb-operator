// +build !ignore_autogenerated

/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1beta1 "k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerSpec) DeepCopyInto(out *ContainerSpec) {
	*out = *in
	if in.Requests != nil {
		in, out := &in.Requests, &out.Requests
		*out = new(ResourceRequirement)
		**out = **in
	}
	if in.Limits != nil {
		in, out := &in.Limits, &out.Limits
		*out = new(ResourceRequirement)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerSpec.
func (in *ContainerSpec) DeepCopy() *ContainerSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PDMember) DeepCopyInto(out *PDMember) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PDMember.
func (in *PDMember) DeepCopy() *PDMember {
	if in == nil {
		return nil
	}
	out := new(PDMember)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PDSpec) DeepCopyInto(out *PDSpec) {
	*out = *in
	in.ContainerSpec.DeepCopyInto(&out.ContainerSpec)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PDSpec.
func (in *PDSpec) DeepCopy() *PDSpec {
	if in == nil {
		return nil
	}
	out := new(PDSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PDStatus) DeepCopyInto(out *PDStatus) {
	*out = *in
	if in.StatefulSet != nil {
		in, out := &in.StatefulSet, &out.StatefulSet
		*out = new(v1beta1.StatefulSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.Members != nil {
		in, out := &in.Members, &out.Members
		*out = make(map[string]PDMember, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PDStatus.
func (in *PDStatus) DeepCopy() *PDStatus {
	if in == nil {
		return nil
	}
	out := new(PDStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceRequirement) DeepCopyInto(out *ResourceRequirement) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRequirement.
func (in *ResourceRequirement) DeepCopy() *ResourceRequirement {
	if in == nil {
		return nil
	}
	out := new(ResourceRequirement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Service) DeepCopyInto(out *Service) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Service.
func (in *Service) DeepCopy() *Service {
	if in == nil {
		return nil
	}
	out := new(Service)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBFailureMember) DeepCopyInto(out *TiDBFailureMember) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBFailureMember.
func (in *TiDBFailureMember) DeepCopy() *TiDBFailureMember {
	if in == nil {
		return nil
	}
	out := new(TiDBFailureMember)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBMember) DeepCopyInto(out *TiDBMember) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBMember.
func (in *TiDBMember) DeepCopy() *TiDBMember {
	if in == nil {
		return nil
	}
	out := new(TiDBMember)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBSpec) DeepCopyInto(out *TiDBSpec) {
	*out = *in
	in.ContainerSpec.DeepCopyInto(&out.ContainerSpec)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBSpec.
func (in *TiDBSpec) DeepCopy() *TiDBSpec {
	if in == nil {
		return nil
	}
	out := new(TiDBSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBStatus) DeepCopyInto(out *TiDBStatus) {
	*out = *in
	if in.StatefulSet != nil {
		in, out := &in.StatefulSet, &out.StatefulSet
		*out = new(v1beta1.StatefulSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.Members != nil {
		in, out := &in.Members, &out.Members
		*out = make(map[string]TiDBMember, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.FailureMembers != nil {
		in, out := &in.FailureMembers, &out.FailureMembers
		*out = make(map[string]TiDBFailureMember, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBStatus.
func (in *TiDBStatus) DeepCopy() *TiDBStatus {
	if in == nil {
		return nil
	}
	out := new(TiDBStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiKVPromGatewaySpec) DeepCopyInto(out *TiKVPromGatewaySpec) {
	*out = *in
	in.ContainerSpec.DeepCopyInto(&out.ContainerSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiKVPromGatewaySpec.
func (in *TiKVPromGatewaySpec) DeepCopy() *TiKVPromGatewaySpec {
	if in == nil {
		return nil
	}
	out := new(TiKVPromGatewaySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiKVSpec) DeepCopyInto(out *TiKVSpec) {
	*out = *in
	in.ContainerSpec.DeepCopyInto(&out.ContainerSpec)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiKVSpec.
func (in *TiKVSpec) DeepCopy() *TiKVSpec {
	if in == nil {
		return nil
	}
	out := new(TiKVSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiKVStatus) DeepCopyInto(out *TiKVStatus) {
	*out = *in
	if in.StatefulSet != nil {
		in, out := &in.StatefulSet, &out.StatefulSet
		*out = new(v1beta1.StatefulSetStatus)
		(*in).DeepCopyInto(*out)
	}
	in.Stores.DeepCopyInto(&out.Stores)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiKVStatus.
func (in *TiKVStatus) DeepCopy() *TiKVStatus {
	if in == nil {
		return nil
	}
	out := new(TiKVStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiKVStore) DeepCopyInto(out *TiKVStore) {
	*out = *in
	in.LastHeartbeatTime.DeepCopyInto(&out.LastHeartbeatTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiKVStore.
func (in *TiKVStore) DeepCopy() *TiKVStore {
	if in == nil {
		return nil
	}
	out := new(TiKVStore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiKVStores) DeepCopyInto(out *TiKVStores) {
	*out = *in
	if in.CurrentStores != nil {
		in, out := &in.CurrentStores, &out.CurrentStores
		*out = make(map[string]TiKVStore, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.TombStoneStores != nil {
		in, out := &in.TombStoneStores, &out.TombStoneStores
		*out = make(map[string]TiKVStore, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiKVStores.
func (in *TiKVStores) DeepCopy() *TiKVStores {
	if in == nil {
		return nil
	}
	out := new(TiKVStores)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TidbCluster) DeepCopyInto(out *TidbCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TidbCluster.
func (in *TidbCluster) DeepCopy() *TidbCluster {
	if in == nil {
		return nil
	}
	out := new(TidbCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TidbCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TidbClusterList) DeepCopyInto(out *TidbClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TidbCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TidbClusterList.
func (in *TidbClusterList) DeepCopy() *TidbClusterList {
	if in == nil {
		return nil
	}
	out := new(TidbClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TidbClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TidbClusterSpec) DeepCopyInto(out *TidbClusterSpec) {
	*out = *in
	in.PD.DeepCopyInto(&out.PD)
	in.TiDB.DeepCopyInto(&out.TiDB)
	in.TiKV.DeepCopyInto(&out.TiKV)
	in.TiKVPromGateway.DeepCopyInto(&out.TiKVPromGateway)
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make([]Service, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TidbClusterSpec.
func (in *TidbClusterSpec) DeepCopy() *TidbClusterSpec {
	if in == nil {
		return nil
	}
	out := new(TidbClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TidbClusterStatus) DeepCopyInto(out *TidbClusterStatus) {
	*out = *in
	in.PD.DeepCopyInto(&out.PD)
	in.TiKV.DeepCopyInto(&out.TiKV)
	in.TiDB.DeepCopyInto(&out.TiDB)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TidbClusterStatus.
func (in *TidbClusterStatus) DeepCopy() *TidbClusterStatus {
	if in == nil {
		return nil
	}
	out := new(TidbClusterStatus)
	in.DeepCopyInto(out)
	return out
}
