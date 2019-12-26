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
	"fmt"
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func GetOrdinalFromPodName(podName string) (int32, error) {
	ordinalStr := podName[strings.LastIndex(podName, "-")+1:]
	ordinalInt, err := strconv.Atoi(ordinalStr)
	if err != nil {
		return int32(0), err
	}
	return int32(ordinalInt), nil
}

func GetNextOrdinalPodName(podName string, ordinal int32) string {
	basicStr := podName[:strings.LastIndex(podName, "-")]
	return fmt.Sprintf("%s-%d", basicStr, ordinal+1)
}

func IsPodOrdinalNotExceedReplicas(pod *corev1.Pod, specReplicas int32) (bool, error) {
	ordinal, err := GetOrdinalFromPodName(pod.Name)
	if err != nil {
		return false, err
	}
	if ordinal < specReplicas {
		return true, nil
	}
	return false, nil
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
