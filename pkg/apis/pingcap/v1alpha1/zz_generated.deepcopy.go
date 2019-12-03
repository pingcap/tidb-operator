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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	json "github.com/pingcap/tidb-operator/pkg/util/json"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Backup) DeepCopyInto(out *Backup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Backup.
func (in *Backup) DeepCopy() *Backup {
	if in == nil {
		return nil
	}
	out := new(Backup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Backup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupCondition) DeepCopyInto(out *BackupCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupCondition.
func (in *BackupCondition) DeepCopy() *BackupCondition {
	if in == nil {
		return nil
	}
	out := new(BackupCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupList) DeepCopyInto(out *BackupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Backup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupList.
func (in *BackupList) DeepCopy() *BackupList {
	if in == nil {
		return nil
	}
	out := new(BackupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BackupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupSchedule) DeepCopyInto(out *BackupSchedule) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupSchedule.
func (in *BackupSchedule) DeepCopy() *BackupSchedule {
	if in == nil {
		return nil
	}
	out := new(BackupSchedule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BackupSchedule) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupScheduleList) DeepCopyInto(out *BackupScheduleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BackupSchedule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupScheduleList.
func (in *BackupScheduleList) DeepCopy() *BackupScheduleList {
	if in == nil {
		return nil
	}
	out := new(BackupScheduleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BackupScheduleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupScheduleSpec) DeepCopyInto(out *BackupScheduleSpec) {
	*out = *in
	if in.MaxBackups != nil {
		in, out := &in.MaxBackups, &out.MaxBackups
		*out = new(int32)
		**out = **in
	}
	if in.MaxReservedTime != nil {
		in, out := &in.MaxReservedTime, &out.MaxReservedTime
		*out = new(string)
		**out = **in
	}
	in.BackupTemplate.DeepCopyInto(&out.BackupTemplate)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupScheduleSpec.
func (in *BackupScheduleSpec) DeepCopy() *BackupScheduleSpec {
	if in == nil {
		return nil
	}
	out := new(BackupScheduleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupScheduleStatus) DeepCopyInto(out *BackupScheduleStatus) {
	*out = *in
	if in.LastBackupTime != nil {
		in, out := &in.LastBackupTime, &out.LastBackupTime
		*out = (*in).DeepCopy()
	}
	if in.AllBackupCleanTime != nil {
		in, out := &in.AllBackupCleanTime, &out.AllBackupCleanTime
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupScheduleStatus.
func (in *BackupScheduleStatus) DeepCopy() *BackupScheduleStatus {
	if in == nil {
		return nil
	}
	out := new(BackupScheduleStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupSpec) DeepCopyInto(out *BackupSpec) {
	*out = *in
	out.From = in.From
	in.StorageProvider.DeepCopyInto(&out.StorageProvider)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupSpec.
func (in *BackupSpec) DeepCopy() *BackupSpec {
	if in == nil {
		return nil
	}
	out := new(BackupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupStatus) DeepCopyInto(out *BackupStatus) {
	*out = *in
	in.TimeStarted.DeepCopyInto(&out.TimeStarted)
	in.TimeCompleted.DeepCopyInto(&out.TimeCompleted)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]BackupCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupStatus.
func (in *BackupStatus) DeepCopy() *BackupStatus {
	if in == nil {
		return nil
	}
	out := new(BackupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentSpec) DeepCopyInto(out *ComponentSpec) {
	*out = *in
	if in.ImagePullPolicy != nil {
		in, out := &in.ImagePullPolicy, &out.ImagePullPolicy
		*out = new(v1.PullPolicy)
		**out = **in
	}
	if in.HostNetwork != nil {
		in, out := &in.HostNetwork, &out.HostNetwork
		*out = new(bool)
		**out = **in
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
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
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentSpec.
func (in *ComponentSpec) DeepCopy() *ComponentSpec {
	if in == nil {
		return nil
	}
	out := new(ComponentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CrdKind) DeepCopyInto(out *CrdKind) {
	*out = *in
	if in.ShortNames != nil {
		in, out := &in.ShortNames, &out.ShortNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AdditionalPrinterColums != nil {
		in, out := &in.AdditionalPrinterColums, &out.AdditionalPrinterColums
		*out = make([]v1beta1.CustomResourceColumnDefinition, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CrdKind.
func (in *CrdKind) DeepCopy() *CrdKind {
	if in == nil {
		return nil
	}
	out := new(CrdKind)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CrdKinds) DeepCopyInto(out *CrdKinds) {
	*out = *in
	in.TiDBCluster.DeepCopyInto(&out.TiDBCluster)
	in.Backup.DeepCopyInto(&out.Backup)
	in.Restore.DeepCopyInto(&out.Restore)
	in.BackupSchedule.DeepCopyInto(&out.BackupSchedule)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CrdKinds.
func (in *CrdKinds) DeepCopy() *CrdKinds {
	if in == nil {
		return nil
	}
	out := new(CrdKinds)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataResource) DeepCopyInto(out *DataResource) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataResource.
func (in *DataResource) DeepCopy() *DataResource {
	if in == nil {
		return nil
	}
	out := new(DataResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DataResource) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataResourceList) DeepCopyInto(out *DataResourceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DataResource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataResourceList.
func (in *DataResourceList) DeepCopy() *DataResourceList {
	if in == nil {
		return nil
	}
	out := new(DataResourceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DataResourceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GcsStorageProvider) DeepCopyInto(out *GcsStorageProvider) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GcsStorageProvider.
func (in *GcsStorageProvider) DeepCopy() *GcsStorageProvider {
	if in == nil {
		return nil
	}
	out := new(GcsStorageProvider)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelperSpec) DeepCopyInto(out *HelperSpec) {
	*out = *in
	if in.ImagePullPolicy != nil {
		in, out := &in.ImagePullPolicy, &out.ImagePullPolicy
		*out = new(v1.PullPolicy)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelperSpec.
func (in *HelperSpec) DeepCopy() *HelperSpec {
	if in == nil {
		return nil
	}
	out := new(HelperSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PDFailureMember) DeepCopyInto(out *PDFailureMember) {
	*out = *in
	in.CreatedAt.DeepCopyInto(&out.CreatedAt)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PDFailureMember.
func (in *PDFailureMember) DeepCopy() *PDFailureMember {
	if in == nil {
		return nil
	}
	out := new(PDFailureMember)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PDMember) DeepCopyInto(out *PDMember) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
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
	in.ComponentSpec.DeepCopyInto(&out.ComponentSpec)
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]json.JsonObject, len(*in))
		for key, val := range *in {
			if val == nil {
				(*out)[key] = nil
			} else {
				(*out)[key] = val.DeepCopyJsonObject()
			}
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
		*out = new(appsv1.StatefulSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.Members != nil {
		in, out := &in.Members, &out.Members
		*out = make(map[string]PDMember, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	in.Leader.DeepCopyInto(&out.Leader)
	if in.FailureMembers != nil {
		in, out := &in.FailureMembers, &out.FailureMembers
		*out = make(map[string]PDFailureMember, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
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
func (in *PumpSpec) DeepCopyInto(out *PumpSpec) {
	*out = *in
	in.ComponentSpec.DeepCopyInto(&out.ComponentSpec)
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]json.JsonObject, len(*in))
		for key, val := range *in {
			if val == nil {
				(*out)[key] = nil
			} else {
				(*out)[key] = val.DeepCopyJsonObject()
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PumpSpec.
func (in *PumpSpec) DeepCopy() *PumpSpec {
	if in == nil {
		return nil
	}
	out := new(PumpSpec)
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
func (in *Resources) DeepCopyInto(out *Resources) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resources.
func (in *Resources) DeepCopy() *Resources {
	if in == nil {
		return nil
	}
	out := new(Resources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Restore) DeepCopyInto(out *Restore) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Restore.
func (in *Restore) DeepCopy() *Restore {
	if in == nil {
		return nil
	}
	out := new(Restore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Restore) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestoreCondition) DeepCopyInto(out *RestoreCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestoreCondition.
func (in *RestoreCondition) DeepCopy() *RestoreCondition {
	if in == nil {
		return nil
	}
	out := new(RestoreCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestoreList) DeepCopyInto(out *RestoreList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Restore, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestoreList.
func (in *RestoreList) DeepCopy() *RestoreList {
	if in == nil {
		return nil
	}
	out := new(RestoreList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RestoreList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestoreSpec) DeepCopyInto(out *RestoreSpec) {
	*out = *in
	out.To = in.To
	in.StorageProvider.DeepCopyInto(&out.StorageProvider)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestoreSpec.
func (in *RestoreSpec) DeepCopy() *RestoreSpec {
	if in == nil {
		return nil
	}
	out := new(RestoreSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RestoreStatus) DeepCopyInto(out *RestoreStatus) {
	*out = *in
	in.TimeStarted.DeepCopyInto(&out.TimeStarted)
	in.TimeCompleted.DeepCopyInto(&out.TimeCompleted)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]RestoreCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RestoreStatus.
func (in *RestoreStatus) DeepCopy() *RestoreStatus {
	if in == nil {
		return nil
	}
	out := new(RestoreStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *S3StorageProvider) DeepCopyInto(out *S3StorageProvider) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new S3StorageProvider.
func (in *S3StorageProvider) DeepCopy() *S3StorageProvider {
	if in == nil {
		return nil
	}
	out := new(S3StorageProvider)
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
func (in *ServiceSpec) DeepCopyInto(out *ServiceSpec) {
	*out = *in
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceSpec.
func (in *ServiceSpec) DeepCopy() *ServiceSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageProvider) DeepCopyInto(out *StorageProvider) {
	*out = *in
	if in.S3 != nil {
		in, out := &in.S3, &out.S3
		*out = new(S3StorageProvider)
		**out = **in
	}
	if in.Gcs != nil {
		in, out := &in.Gcs, &out.Gcs
		*out = new(GcsStorageProvider)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageProvider.
func (in *StorageProvider) DeepCopy() *StorageProvider {
	if in == nil {
		return nil
	}
	out := new(StorageProvider)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBAccessConfig) DeepCopyInto(out *TiDBAccessConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBAccessConfig.
func (in *TiDBAccessConfig) DeepCopy() *TiDBAccessConfig {
	if in == nil {
		return nil
	}
	out := new(TiDBAccessConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBFailureMember) DeepCopyInto(out *TiDBFailureMember) {
	*out = *in
	in.CreatedAt.DeepCopyInto(&out.CreatedAt)
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
func (in *TiDBServiceSpec) DeepCopyInto(out *TiDBServiceSpec) {
	*out = *in
	in.ServiceSpec.DeepCopyInto(&out.ServiceSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBServiceSpec.
func (in *TiDBServiceSpec) DeepCopy() *TiDBServiceSpec {
	if in == nil {
		return nil
	}
	out := new(TiDBServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBSlowLogTailerSpec) DeepCopyInto(out *TiDBSlowLogTailerSpec) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
	if in.ImagePullPolicy != nil {
		in, out := &in.ImagePullPolicy, &out.ImagePullPolicy
		*out = new(v1.PullPolicy)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiDBSlowLogTailerSpec.
func (in *TiDBSlowLogTailerSpec) DeepCopy() *TiDBSlowLogTailerSpec {
	if in == nil {
		return nil
	}
	out := new(TiDBSlowLogTailerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiDBSpec) DeepCopyInto(out *TiDBSpec) {
	*out = *in
	in.ComponentSpec.DeepCopyInto(&out.ComponentSpec)
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(TiDBServiceSpec)
		(*in).DeepCopyInto(*out)
	}
	in.SlowLogTailer.DeepCopyInto(&out.SlowLogTailer)
	if in.Plugins != nil {
		in, out := &in.Plugins, &out.Plugins
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]json.JsonObject, len(*in))
		for key, val := range *in {
			if val == nil {
				(*out)[key] = nil
			} else {
				(*out)[key] = val.DeepCopyJsonObject()
			}
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
		*out = new(appsv1.StatefulSetStatus)
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
			(*out)[key] = *val.DeepCopy()
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
func (in *TiKVFailureStore) DeepCopyInto(out *TiKVFailureStore) {
	*out = *in
	in.CreatedAt.DeepCopyInto(&out.CreatedAt)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TiKVFailureStore.
func (in *TiKVFailureStore) DeepCopy() *TiKVFailureStore {
	if in == nil {
		return nil
	}
	out := new(TiKVFailureStore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TiKVSpec) DeepCopyInto(out *TiKVSpec) {
	*out = *in
	in.ComponentSpec.DeepCopyInto(&out.ComponentSpec)
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = new(ServiceSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]json.JsonObject, len(*in))
		for key, val := range *in {
			if val == nil {
				(*out)[key] = nil
			} else {
				(*out)[key] = val.DeepCopyJsonObject()
			}
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
		*out = new(appsv1.StatefulSetStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.Stores != nil {
		in, out := &in.Stores, &out.Stores
		*out = make(map[string]TiKVStore, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.TombstoneStores != nil {
		in, out := &in.TombstoneStores, &out.TombstoneStores
		*out = make(map[string]TiKVStore, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.FailureStores != nil {
		in, out := &in.FailureStores, &out.FailureStores
		*out = make(map[string]TiKVFailureStore, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
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
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
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
	in.ListMeta.DeepCopyInto(&out.ListMeta)
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
	if in.Pump != nil {
		in, out := &in.Pump, &out.Pump
		*out = new(PumpSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Helper.DeepCopyInto(&out.Helper)
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make([]Service, len(*in))
		copy(*out, *in)
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
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
