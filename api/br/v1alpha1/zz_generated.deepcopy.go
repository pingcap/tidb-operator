//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzblobStorageProvider) DeepCopyInto(out *AzblobStorageProvider) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzblobStorageProvider.
func (in *AzblobStorageProvider) DeepCopy() *AzblobStorageProvider {
	if in == nil {
		return nil
	}
	out := new(AzblobStorageProvider)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BRConfig) DeepCopyInto(out *BRConfig) {
	*out = *in
	if in.Concurrency != nil {
		in, out := &in.Concurrency, &out.Concurrency
		*out = new(uint32)
		**out = **in
	}
	if in.RateLimit != nil {
		in, out := &in.RateLimit, &out.RateLimit
		*out = new(uint)
		**out = **in
	}
	if in.Checksum != nil {
		in, out := &in.Checksum, &out.Checksum
		*out = new(bool)
		**out = **in
	}
	if in.CheckRequirements != nil {
		in, out := &in.CheckRequirements, &out.CheckRequirements
		*out = new(bool)
		**out = **in
	}
	if in.SendCredToTikv != nil {
		in, out := &in.SendCredToTikv, &out.SendCredToTikv
		*out = new(bool)
		**out = **in
	}
	if in.OnLine != nil {
		in, out := &in.OnLine, &out.OnLine
		*out = new(bool)
		**out = **in
	}
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BRConfig.
func (in *BRConfig) DeepCopy() *BRConfig {
	if in == nil {
		return nil
	}
	out := new(BRConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackoffRetryPolicy) DeepCopyInto(out *BackoffRetryPolicy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackoffRetryPolicy.
func (in *BackoffRetryPolicy) DeepCopy() *BackoffRetryPolicy {
	if in == nil {
		return nil
	}
	out := new(BackoffRetryPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackoffRetryRecord) DeepCopyInto(out *BackoffRetryRecord) {
	*out = *in
	if in.DetectFailedAt != nil {
		in, out := &in.DetectFailedAt, &out.DetectFailedAt
		*out = (*in).DeepCopy()
	}
	if in.ExpectedRetryAt != nil {
		in, out := &in.ExpectedRetryAt, &out.ExpectedRetryAt
		*out = (*in).DeepCopy()
	}
	if in.RealRetryAt != nil {
		in, out := &in.RealRetryAt, &out.RealRetryAt
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackoffRetryRecord.
func (in *BackoffRetryRecord) DeepCopy() *BackoffRetryRecord {
	if in == nil {
		return nil
	}
	out := new(BackoffRetryRecord)
	in.DeepCopyInto(out)
	return out
}

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
	in.Condition.DeepCopyInto(&out.Condition)
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
	if in.CompactSpan != nil {
		in, out := &in.CompactSpan, &out.CompactSpan
		*out = new(string)
		**out = **in
	}
	in.BackupTemplate.DeepCopyInto(&out.BackupTemplate)
	if in.LogBackupTemplate != nil {
		in, out := &in.LogBackupTemplate, &out.LogBackupTemplate
		*out = new(BackupSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.CompactBackupTemplate != nil {
		in, out := &in.CompactBackupTemplate, &out.CompactBackupTemplate
		*out = new(CompactSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.StorageClassName != nil {
		in, out := &in.StorageClassName, &out.StorageClassName
		*out = new(string)
		**out = **in
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.BR != nil {
		in, out := &in.BR, &out.BR
		*out = new(BRConfig)
		(*in).DeepCopyInto(*out)
	}
	in.StorageProvider.DeepCopyInto(&out.StorageProvider)
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
	if in.LogBackup != nil {
		in, out := &in.LogBackup, &out.LogBackup
		*out = new(string)
		**out = **in
	}
	if in.LogBackupStartTs != nil {
		in, out := &in.LogBackupStartTs, &out.LogBackupStartTs
		*out = (*in).DeepCopy()
	}
	if in.LastBackupTime != nil {
		in, out := &in.LastBackupTime, &out.LastBackupTime
		*out = (*in).DeepCopy()
	}
	if in.LastCompactTs != nil {
		in, out := &in.LastCompactTs, &out.LastCompactTs
		*out = (*in).DeepCopy()
	}
	if in.NextCompactEndTs != nil {
		in, out := &in.NextCompactEndTs, &out.NextCompactEndTs
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
	in.ResourceRequirements.DeepCopyInto(&out.ResourceRequirements)
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TikvGCLifeTime != nil {
		in, out := &in.TikvGCLifeTime, &out.TikvGCLifeTime
		*out = new(string)
		**out = **in
	}
	in.StorageProvider.DeepCopyInto(&out.StorageProvider)
	if in.StorageClassName != nil {
		in, out := &in.StorageClassName, &out.StorageClassName
		*out = new(string)
		**out = **in
	}
	if in.BR != nil {
		in, out := &in.BR, &out.BR
		*out = new(BRConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Dumpling != nil {
		in, out := &in.Dumpling, &out.Dumpling
		*out = new(DumplingConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.TableFilter != nil {
		in, out := &in.TableFilter, &out.TableFilter
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.CleanOption != nil {
		in, out := &in.CleanOption, &out.CleanOption
		*out = new(CleanOption)
		**out = **in
	}
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	out.BackoffRetryPolicy = in.BackoffRetryPolicy
	if in.AdditionalVolumes != nil {
		in, out := &in.AdditionalVolumes, &out.AdditionalVolumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AdditionalVolumeMounts != nil {
		in, out := &in.AdditionalVolumeMounts, &out.AdditionalVolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
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
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.LogSubCommandStatuses != nil {
		in, out := &in.LogSubCommandStatuses, &out.LogSubCommandStatuses
		*out = make(map[LogSubCommandType]LogSubCommandStatus, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Progresses != nil {
		in, out := &in.Progresses, &out.Progresses
		*out = make([]Progress, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.BackoffRetryStatus != nil {
		in, out := &in.BackoffRetryStatus, &out.BackoffRetryStatus
		*out = make([]BackoffRetryRecord, len(*in))
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
func (in *BatchDeleteOption) DeepCopyInto(out *BatchDeleteOption) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BatchDeleteOption.
func (in *BatchDeleteOption) DeepCopy() *BatchDeleteOption {
	if in == nil {
		return nil
	}
	out := new(BatchDeleteOption)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CleanOption) DeepCopyInto(out *CleanOption) {
	*out = *in
	out.BatchDeleteOption = in.BatchDeleteOption
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CleanOption.
func (in *CleanOption) DeepCopy() *CleanOption {
	if in == nil {
		return nil
	}
	out := new(CleanOption)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompactBackup) DeepCopyInto(out *CompactBackup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompactBackup.
func (in *CompactBackup) DeepCopy() *CompactBackup {
	if in == nil {
		return nil
	}
	out := new(CompactBackup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CompactBackup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompactBackupList) DeepCopyInto(out *CompactBackupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CompactBackup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompactBackupList.
func (in *CompactBackupList) DeepCopy() *CompactBackupList {
	if in == nil {
		return nil
	}
	out := new(CompactBackupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CompactBackupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompactRetryRecord) DeepCopyInto(out *CompactRetryRecord) {
	*out = *in
	in.DetectFailedAt.DeepCopyInto(&out.DetectFailedAt)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompactRetryRecord.
func (in *CompactRetryRecord) DeepCopy() *CompactRetryRecord {
	if in == nil {
		return nil
	}
	out := new(CompactRetryRecord)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompactSpec) DeepCopyInto(out *CompactSpec) {
	*out = *in
	in.ResourceRequirements.DeepCopyInto(&out.ResourceRequirements)
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.StorageProvider.DeepCopyInto(&out.StorageProvider)
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.BR != nil {
		in, out := &in.BR, &out.BR
		*out = new(BRConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.AdditionalVolumes != nil {
		in, out := &in.AdditionalVolumes, &out.AdditionalVolumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AdditionalVolumeMounts != nil {
		in, out := &in.AdditionalVolumeMounts, &out.AdditionalVolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompactSpec.
func (in *CompactSpec) DeepCopy() *CompactSpec {
	if in == nil {
		return nil
	}
	out := new(CompactSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CompactStatus) DeepCopyInto(out *CompactStatus) {
	*out = *in
	if in.RetryStatus != nil {
		in, out := &in.RetryStatus, &out.RetryStatus
		*out = make([]CompactRetryRecord, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CompactStatus.
func (in *CompactStatus) DeepCopy() *CompactStatus {
	if in == nil {
		return nil
	}
	out := new(CompactStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DumplingConfig) DeepCopyInto(out *DumplingConfig) {
	*out = *in
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TableFilter != nil {
		in, out := &in.TableFilter, &out.TableFilter
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DumplingConfig.
func (in *DumplingConfig) DeepCopy() *DumplingConfig {
	if in == nil {
		return nil
	}
	out := new(DumplingConfig)
	in.DeepCopyInto(out)
	return out
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
func (in *LocalStorageProvider) DeepCopyInto(out *LocalStorageProvider) {
	*out = *in
	in.Volume.DeepCopyInto(&out.Volume)
	in.VolumeMount.DeepCopyInto(&out.VolumeMount)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LocalStorageProvider.
func (in *LocalStorageProvider) DeepCopy() *LocalStorageProvider {
	if in == nil {
		return nil
	}
	out := new(LocalStorageProvider)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogSubCommandStatus) DeepCopyInto(out *LogSubCommandStatus) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogSubCommandStatus.
func (in *LogSubCommandStatus) DeepCopy() *LogSubCommandStatus {
	if in == nil {
		return nil
	}
	out := new(LogSubCommandStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Progress) DeepCopyInto(out *Progress) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Progress.
func (in *Progress) DeepCopy() *Progress {
	if in == nil {
		return nil
	}
	out := new(Progress)
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
	in.ResourceRequirements.DeepCopyInto(&out.ResourceRequirements)
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.To != nil {
		in, out := &in.To, &out.To
		*out = new(TiDBAccessConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.TikvGCLifeTime != nil {
		in, out := &in.TikvGCLifeTime, &out.TikvGCLifeTime
		*out = new(string)
		**out = **in
	}
	in.StorageProvider.DeepCopyInto(&out.StorageProvider)
	in.PitrFullBackupStorageProvider.DeepCopyInto(&out.PitrFullBackupStorageProvider)
	if in.StorageClassName != nil {
		in, out := &in.StorageClassName, &out.StorageClassName
		*out = new(string)
		**out = **in
	}
	if in.BR != nil {
		in, out := &in.BR, &out.BR
		*out = new(BRConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.TableFilter != nil {
		in, out := &in.TableFilter, &out.TableFilter
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PodSecurityContext != nil {
		in, out := &in.PodSecurityContext, &out.PodSecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.AdditionalVolumes != nil {
		in, out := &in.AdditionalVolumes, &out.AdditionalVolumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AdditionalVolumeMounts != nil {
		in, out := &in.AdditionalVolumeMounts, &out.AdditionalVolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
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
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Progresses != nil {
		in, out := &in.Progresses, &out.Progresses
		*out = make([]Progress, len(*in))
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
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
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
func (in *StorageProvider) DeepCopyInto(out *StorageProvider) {
	*out = *in
	if in.S3 != nil {
		in, out := &in.S3, &out.S3
		*out = new(S3StorageProvider)
		(*in).DeepCopyInto(*out)
	}
	if in.Gcs != nil {
		in, out := &in.Gcs, &out.Gcs
		*out = new(GcsStorageProvider)
		**out = **in
	}
	if in.Azblob != nil {
		in, out := &in.Azblob, &out.Azblob
		*out = new(AzblobStorageProvider)
		**out = **in
	}
	if in.Local != nil {
		in, out := &in.Local, &out.Local
		*out = new(LocalStorageProvider)
		(*in).DeepCopyInto(*out)
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
	if in.TLSClientSecretName != nil {
		in, out := &in.TLSClientSecretName, &out.TLSClientSecretName
		*out = new(string)
		**out = **in
	}
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
