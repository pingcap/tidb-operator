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
	"fmt"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

// HashContents hashes the contents using FNV hashing. The returned hash will be a safe encoded string to avoid bad words.
func HashContents(contents []byte) string {
	hf := fnv.New32()
	if len(contents) > 0 {
		hf.Write(contents)
	}
	return rand.SafeEncodeString(fmt.Sprint(hf.Sum32()))
}

// GetTidbPort return the tidb port
func (tac *TiDBAccessConfig) GetTidbPort() int32 {
	if tac.Port != 0 {
		return tac.Port
	}
	return DefaultTiDBServicePort
}

// GetTidbUser return the tidb user
func (tac *TiDBAccessConfig) GetTidbUser() string {
	if len(tac.User) != 0 {
		return tac.User
	}
	return DefaultTidbUser
}

// GetTidbEndpoint return the tidb endpoint for access tidb cluster directly
func (tac *TiDBAccessConfig) GetTidbEndpoint() string {
	return fmt.Sprintf("%s_%d", tac.Host, tac.GetTidbPort())
}

// workaround for removing dependency about package 'advanced-statefulset'
// copy from https://github.com/pingcap/advanced-statefulset/blob/f94e356d25058396e94d33c3fe7224d5a2ca1517/client/apis/apps/v1/helper/helper.go#L94
func GetPodOrdinalsFromReplicasAndDeleteSlots(replicas int32, deleteSlots sets.Int32) sets.Int32 {
	maxReplicaCount, deleteSlots := GetMaxReplicaCountAndDeleteSlots(replicas, deleteSlots)
	podOrdinals := sets.NewInt32()
	for i := int32(0); i < maxReplicaCount; i++ {
		if !deleteSlots.Has(i) {
			podOrdinals.Insert(i)
		}
	}
	return podOrdinals
}

// GetMaxReplicaCountAndDeleteSlots returns the max replica count and delete
// slots. The desired slots of this stateful set will be [0, replicaCount) - [delete slots].
//
// workaround for removing dependency about package 'advanced-statefulset'
// copy from https://github.com/pingcap/advanced-statefulset/blob/f94e356d25058396e94d33c3fe7224d5a2ca1517/client/apis/apps/v1/helper/helper.go#L74
func GetMaxReplicaCountAndDeleteSlots(replicas int32, deleteSlots sets.Int32) (int32, sets.Int32) {
	replicaCount := replicas
	deleteSlotsCopy := sets.NewInt32()
	for k := range deleteSlots {
		deleteSlotsCopy.Insert(k)
	}
	for _, deleteSlot := range deleteSlotsCopy.List() {
		if deleteSlot < replicaCount {
			replicaCount++
		} else {
			deleteSlotsCopy.Delete(deleteSlot)
		}
	}
	return replicaCount, deleteSlotsCopy
}

// GetStorageVolumeName return the storage volume name for a component's storage volume (not support TiFlash).
//
// When storageVolumeName is empty, it indicate volume is base data volume which have special name.
// When storageVolumeName is not empty, it indicate volume is additional volume which is declaired in `spec.storageVolumes`.
func GetStorageVolumeName(storageVolumeName string, memberType MemberType) StorageVolumeName {
	if storageVolumeName == "" {
		switch memberType {
		case PumpMemberType:
			return StorageVolumeName("data")
		default:
			return StorageVolumeName(memberType.String())
		}
	}
	return StorageVolumeName(fmt.Sprintf("%s-%s", memberType.String(), storageVolumeName))
}

// GetStorageVolumeNameForTiFlash return the PVC template name for a TiFlash's data volume
func GetStorageVolumeNameForTiFlash(index int) StorageVolumeName {
	return StorageVolumeName(fmt.Sprintf("data%d", index))
}

var (
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
	GetMemberType() MemberType
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
	// GetConditions return `status.conditions`
	//
	// If need to change the condition, please use `SetCondition`
	GetConditions() []metav1.Condition

	// SetCondition sets the corresponding condition in conditions to newCondition.
	// 1. if the condition of the specified type already exists (all fields of the existing condition are updated to
	//    newCondition, LastTransitionTime is set to now if the new status differs from the old status)
	// 2. if a condition of the specified type does not exist (LastTransitionTime is set to now() if unset, and newCondition is appended)
	SetCondition(condition metav1.Condition)
	// RemoveStatusCondition removes the corresponding conditionType from conditions.
	RemoveCondition(conditionType string)
}

func ComponentStatusFromTC(tc *TidbCluster) []ComponentStatus {
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

func ComponentStatusFromDC(dc *DMCluster) []ComponentStatus {
	components := []ComponentStatus{}
	components = append(components, &dc.Status.Master)
	if dc.Spec.Worker != nil {
		components = append(components, &dc.Status.Worker)
	}
	return components
}

func (s *PDStatus) GetMemberType() MemberType {
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

func (s *TiKVStatus) GetMemberType() MemberType {
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

func (s *TiDBStatus) GetMemberType() MemberType {
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

func (s *PumpStatus) GetMemberType() MemberType {
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

func (s *TiFlashStatus) GetMemberType() MemberType {
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

func (s *TiCDCStatus) GetMemberType() MemberType {
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

func (s *MasterStatus) GetMemberType() MemberType {
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

func (s *WorkerStatus) GetMemberType() MemberType {
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
