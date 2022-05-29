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

// GetPVCTemplateName return the PVC template name for a component's storage volume
//
// When storageVolumeName is empty, it indicate volume is base data volume which have special name. (not support TiFlash)
// When storageVolumeName is not empty, it indicate volume is additional volume which is declaired in `spec.storageVolumes`.
func GetPVCTemplateName(storageVolumeName string, memberType MemberType) string {
	if storageVolumeName == "" {
		switch memberType {
		case PumpMemberType:
			return "data"
		default:
			return memberType.String()
		}
	}
	return fmt.Sprintf("%s-%s", memberType.String(), storageVolumeName)
}

// GetPVCTemplateNameForTiFlash return the PVC template name for a TiFlash's data volume
func GetPVCTemplateNameForTiFlash(index int) string {
	return fmt.Sprintf("data%d", index)
}
