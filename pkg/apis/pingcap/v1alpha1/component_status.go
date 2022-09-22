// Copyright 2019 PingCAP, Inc.
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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// supported components
	_ ComponentStatus = &PDStatus{}
	_ ComponentStatus = &TiKVStatus{}
	_ ComponentStatus = &TiDBStatus{}
	_ ComponentStatus = &PumpStatus{}
	_ ComponentStatus = &TiFlashStatus{}
	_ ComponentStatus = &TiCDCStatus{}
	_ ComponentStatus = &MasterStatus{}
	_ ComponentStatus = &WorkerStatus{}
)

type ComponentStatus interface {
	MemberType() MemberType
	// GetSynced returns `status.synced`
	//
	// For tidb and pump, it is always true.
	GetSynced() bool
	// GetSynced returns `status.phase`
	GetPhase() MemberPhase
	// GetVolumes return `status.volumes`
	//
	// NOTE: change the map will modify the status.
	GetVolumes() map[StorageVolumeName]*StorageVolumeStatus
	// GetConditions returns `status.conditions`
	//
	// If need to change the condition, please use `SetCondition`
	GetConditions() []metav1.Condition
	// GetStatefulSet returns `status.statefulset`
	//
	// NOTE: change the return will modify the status.
	GetStatefulSet() *appsv1.StatefulSetStatus

	// SetSynced set the `status.synced` field of the component
	//
	// Not supported for tidb and pump
	SetSynced(bool)
	// SetPhase sets the phase of the component.
	SetPhase(phase MemberPhase)
	// SetVolumes sets the `status.volumes`
	SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus)
	// SetCondition sets the corresponding condition in conditions to newCondition.
	// 1. if the condition of the specified type already exists (all fields of the existing condition are updated to
	//    newCondition, LastTransitionTime is set to now if the new status differs from the old status)
	// 2. if a condition of the specified type does not exist (LastTransitionTime is set to now() if unset, and newCondition is appended)
	SetCondition(condition metav1.Condition)
	// RemoveStatusCondition removes the corresponding conditionType from conditions.
	RemoveCondition(conditionType string)
	// SetStatefulSet sets the `status.statefulset`
	SetStatefulSet(sts *appsv1.StatefulSetStatus)
}

func (tc *TidbCluster) AllComponentStatus() []ComponentStatus {
	components := []ComponentStatus{}
	if tc.Spec.PD != nil {
		components = append(components, &tc.Status.PD)
	}
	if tc.Spec.TiDB != nil {
		components = append(components, &tc.Status.TiDB)
	}
	if tc.Spec.TiKV != nil {
		components = append(components, &tc.Status.TiKV)
	}
	if tc.Spec.TiFlash != nil {
		components = append(components, &tc.Status.TiFlash)
	}
	if tc.Spec.TiCDC != nil {
		components = append(components, &tc.Status.TiCDC)
	}
	if tc.Spec.Pump != nil {
		components = append(components, &tc.Status.Pump)
	}
	return components
}

func (tc *TidbCluster) ComponentStatus(typ MemberType) ComponentStatus {
	components := tc.AllComponentStatus()
	for _, component := range components {
		if component.MemberType() == typ {
			return component
		}
	}
	return nil
}

func (dc *DMCluster) AllComponentStatus() []ComponentStatus {
	components := []ComponentStatus{}
	components = append(components, &dc.Status.Master)
	if dc.Spec.Worker != nil {
		components = append(components, &dc.Status.Worker)
	}
	return components
}

func (dc *DMCluster) ComponentStatus(typ MemberType) ComponentStatus {
	components := dc.AllComponentStatus()
	for _, component := range components {
		if component.MemberType() == typ {
			return component
		}
	}
	return nil
}

func (s *PDStatus) MemberType() MemberType {
	return PDMemberType
}
func (s *PDStatus) GetSynced() bool {
	return s.Synced
}
func (s *PDStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *PDStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *PDStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *PDStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *PDStatus) SetSynced(synced bool) {
	s.Synced = synced
}
func (s *PDStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *PDStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *PDStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *PDStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *PDStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}

func (s *TiKVStatus) MemberType() MemberType {
	return TiKVMemberType
}
func (s *TiKVStatus) GetSynced() bool {
	return s.Synced
}
func (s *TiKVStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *TiKVStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *TiKVStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *TiKVStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *TiKVStatus) SetSynced(synced bool) {
	s.Synced = synced
}
func (s *TiKVStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *TiKVStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *TiKVStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *TiKVStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *TiKVStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}

func (s *TiDBStatus) MemberType() MemberType {
	return TiDBMemberType
}
func (s *TiDBStatus) GetSynced() bool {
	return true
}
func (s *TiDBStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *TiDBStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *TiDBStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *TiDBStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *TiDBStatus) SetSynced(synced bool) {}
func (s *TiDBStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *TiDBStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *TiDBStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *TiDBStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *TiDBStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}

func (s *PumpStatus) MemberType() MemberType {
	return PumpMemberType
}
func (s *PumpStatus) GetSynced() bool {
	return true
}
func (s *PumpStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *PumpStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *PumpStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *PumpStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *PumpStatus) SetSynced(synced bool) {}
func (s *PumpStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *PumpStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *PumpStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *PumpStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *PumpStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}

func (s *TiFlashStatus) MemberType() MemberType {
	return TiFlashMemberType
}
func (s *TiFlashStatus) GetSynced() bool {
	return s.Synced
}
func (s *TiFlashStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *TiFlashStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *TiFlashStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *TiFlashStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *TiFlashStatus) SetSynced(synced bool) {
	s.Synced = synced
}
func (s *TiFlashStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *TiFlashStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *TiFlashStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *TiFlashStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *TiFlashStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}

func (s *TiCDCStatus) MemberType() MemberType {
	return TiCDCMemberType
}
func (s *TiCDCStatus) GetSynced() bool {
	return s.Synced
}
func (s *TiCDCStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *TiCDCStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *TiCDCStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *TiCDCStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *TiCDCStatus) SetSynced(synced bool) {
	s.Synced = synced
}
func (s *TiCDCStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *TiCDCStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *TiCDCStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *TiCDCStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *TiCDCStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}

func (s *MasterStatus) MemberType() MemberType {
	return DMMasterMemberType
}
func (s *MasterStatus) GetSynced() bool {
	return s.Synced
}
func (s *MasterStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *MasterStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *MasterStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *MasterStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *MasterStatus) SetSynced(synced bool) {
	s.Synced = synced
}
func (s *MasterStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *MasterStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *MasterStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *MasterStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *MasterStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}

func (s *WorkerStatus) MemberType() MemberType {
	return DMWorkerMemberType
}
func (s *WorkerStatus) GetSynced() bool {
	return s.Synced
}
func (s *WorkerStatus) GetPhase() MemberPhase {
	return s.Phase
}
func (s *WorkerStatus) GetVolumes() map[StorageVolumeName]*StorageVolumeStatus {
	return s.Volumes
}
func (s *WorkerStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}
func (s *WorkerStatus) GetStatefulSet() *appsv1.StatefulSetStatus {
	return s.StatefulSet
}
func (s *WorkerStatus) SetSynced(synced bool) {
	s.Synced = synced
}
func (s *WorkerStatus) SetCondition(newCondition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = []metav1.Condition{}
	}
	conditions := s.Conditions
	meta.SetStatusCondition(&conditions, newCondition)
	s.Conditions = conditions
}
func (s *WorkerStatus) RemoveCondition(conditionType string) {
	if s.Conditions == nil {
		return
	}
	conditions := s.Conditions
	meta.RemoveStatusCondition(&conditions, conditionType)
	s.Conditions = conditions
}
func (s *WorkerStatus) SetPhase(phase MemberPhase) {
	s.Phase = phase
}
func (s *WorkerStatus) SetStatefulSet(sts *appsv1.StatefulSetStatus) {
	s.StatefulSet = sts
}
func (s *WorkerStatus) SetVolumes(vols map[StorageVolumeName]*StorageVolumeStatus) {
	s.Volumes = vols
}
