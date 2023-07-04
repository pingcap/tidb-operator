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
	"crypto/md5"
	"fmt"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
	labelLengthLimit = 63
	hashSize         = 8
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
	return DefaultTiDBServerPort
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

func GenValidName(name string) string {
	if len(name) <= labelLengthLimit {
		return name
	}

	prefixLimit := labelLengthLimit - hashSize - 1
	hash := fmt.Sprintf("%x", md5.Sum([]byte(name)))[:hashSize]

	return fmt.Sprintf("%s-%s", name[:prefixLimit], hash)
}
