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

package util

import (
	corev1 "k8s.io/api/core/v1"
)

func GetContainerStatusFromPod(pod *corev1.Pod, cond func(corev1.ContainerStatus) bool) *corev1.ContainerStatus {
	if pod == nil {
		return nil
	}
	for _, c := range pod.Status.ContainerStatuses {
		if cond(c) {
			return &c
		}
	}
	return nil
}
