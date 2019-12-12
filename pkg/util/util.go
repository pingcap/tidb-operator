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
	"k8s.io/apimachinery/pkg/api/resource"
	glog "k8s.io/klog"
)

// ResourceRequirement creates ResourceRequirements for MemberSpec
// Optionally pass in a default value
func ResourceRequirement(resources v1alpha1.Resources, defaultRequests ...corev1.ResourceRequirements) corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}
	if len(defaultRequests) > 0 {
		defaultRequest := defaultRequests[0]
		rr.Requests = make(map[corev1.ResourceName]resource.Quantity)
		rr.Requests[corev1.ResourceCPU] = defaultRequest.Requests[corev1.ResourceCPU]
		rr.Requests[corev1.ResourceMemory] = defaultRequest.Requests[corev1.ResourceMemory]
		rr.Limits = make(map[corev1.ResourceName]resource.Quantity)
		rr.Limits[corev1.ResourceCPU] = defaultRequest.Limits[corev1.ResourceCPU]
		rr.Limits[corev1.ResourceMemory] = defaultRequest.Limits[corev1.ResourceMemory]
	}
	if resources.Requests != nil {
		if rr.Requests == nil {
			rr.Requests = make(map[corev1.ResourceName]resource.Quantity)
		}
		if resources.Requests.CPU != "" {
			if q, err := resource.ParseQuantity(resources.Requests.CPU); err != nil {
				glog.Errorf("failed to parse CPU resource %s to quantity: %v", resources.Requests.CPU, err)
			} else {
				rr.Requests[corev1.ResourceCPU] = q
			}
		}
		if resources.Requests.Memory != "" {
			if q, err := resource.ParseQuantity(resources.Requests.Memory); err != nil {
				glog.Errorf("failed to parse memory resource %s to quantity: %v", resources.Requests.Memory, err)
			} else {
				rr.Requests[corev1.ResourceMemory] = q
			}
		}
	}
	if resources.Limits != nil {
		if rr.Limits == nil {
			rr.Limits = make(map[corev1.ResourceName]resource.Quantity)
		}
		if resources.Limits.CPU != "" {
			if q, err := resource.ParseQuantity(resources.Limits.CPU); err != nil {
				glog.Errorf("failed to parse CPU resource %s to quantity: %v", resources.Limits.CPU, err)
			} else {
				rr.Limits[corev1.ResourceCPU] = q
			}
		}
		if resources.Limits.Memory != "" {
			if q, err := resource.ParseQuantity(resources.Limits.Memory); err != nil {
				glog.Errorf("failed to parse memory resource %s to quantity: %v", resources.Limits.Memory, err)
			} else {
				rr.Limits[corev1.ResourceMemory] = q
			}
		}
	}
	return rr
}

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
