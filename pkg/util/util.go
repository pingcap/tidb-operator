// Copyright 2018 PingCAP, Inc.
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

package util

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func GetOrdinalFromPodName(podName string) (int32, error) {
	ordinalStr := podName[strings.LastIndex(podName, "-")+1:]
	ordinalInt, err := strconv.Atoi(ordinalStr)
	if err != nil {
		return int32(0), err
	}
	return int32(ordinalInt), nil
}

func IsPodOrdinalNotExceedReplicas(pod *corev1.Pod, sts *apps.StatefulSet) (bool, error) {
	ordinal, err := GetOrdinalFromPodName(pod.Name)
	if err != nil {
		return false, err
	}
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		return helper.GetPodOrdinals(*sts.Spec.Replicas, sts).Has(ordinal), nil
	}
	return ordinal < *sts.Spec.Replicas, nil
}

func getDeleteSlots(tc *v1alpha1.TidbCluster, annKey string) (deleteSlots sets.Int32) {
	deleteSlots = sets.NewInt32()
	annotations := tc.GetAnnotations()
	if annotations == nil {
		return
	}
	value, ok := annotations[annKey]
	if !ok {
		return
	}
	var slice []int32
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deleteSlots.Insert(slice...)
	return
}

// GetPodOrdinals gets desired ordials of member in given TidbCluster.
func GetPodOrdinals(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) (sets.Int32, error) {
	var ann string
	var replicas int32
	if memberType == v1alpha1.PDMemberType {
		ann = label.AnnPDDeleteSlots
		replicas = tc.Spec.PD.Replicas
	} else if memberType == v1alpha1.TiKVMemberType {
		ann = label.AnnTiKVDeleteSlots
		replicas = tc.Spec.TiKV.Replicas
	} else if memberType == v1alpha1.TiDBMemberType {
		ann = label.AnnTiDBDeleteSlots
		replicas = tc.Spec.TiDB.Replicas
	} else {
		return nil, fmt.Errorf("unknown member type %v", memberType)
	}
	deleteSlots := getDeleteSlots(tc, ann)
	maxReplicaCount, deleteSlots := helper.GetMaxReplicaCountAndDeleteSlots(replicas, deleteSlots)
	podOrdinals := sets.NewInt32()
	for i := int32(0); i < maxReplicaCount; i++ {
		if !deleteSlots.Has(i) {
			podOrdinals.Insert(i)
		}
	}
	return podOrdinals, nil
}

func OrdinalPVCName(memberType v1alpha1.MemberType, setName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", memberType, setName, ordinal)
}

// IsSubMapOf returns whether the first map is a sub map of the second map
func IsSubMapOf(first map[string]string, second map[string]string) bool {
	for k, v := range first {
		if second == nil {
			return false
		}
		if second[k] != v {
			return false
		}
	}
	return true
}

func GetPodName(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", tc.Name, memberType.String(), ordinal)
}
