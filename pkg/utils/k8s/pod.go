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

package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func GetResourceRequirements(req v1alpha1.ResourceRequirements) corev1.ResourceRequirements {
	if req.CPU == nil && req.Memory == nil {
		return corev1.ResourceRequirements{}
	}
	ret := corev1.ResourceRequirements{
		Limits:   map[corev1.ResourceName]resource.Quantity{},
		Requests: map[corev1.ResourceName]resource.Quantity{},
	}
	if req.CPU != nil {
		ret.Requests[corev1.ResourceCPU] = *req.CPU
		ret.Limits[corev1.ResourceCPU] = *req.CPU
	}
	if req.Memory != nil {
		ret.Requests[corev1.ResourceMemory] = *req.Memory
		ret.Limits[corev1.ResourceMemory] = *req.Memory
	}
	return ret
}

// AnnoProm returns the prometheus annotations for a pod.
func AnnoProm(port int32, path string) map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   fmt.Sprintf("%d", port),
		"prometheus.io/path":   path,
	}
}

// AnnoAdditionalProm returns the additional prometheus annotations for a pod.
// Some pods may have multiple prometheus endpoints.
// We assume the same path is used for all endpoints.
func AnnoAdditionalProm(name string, port int32) map[string]string {
	return map[string]string{
		fmt.Sprintf("%s.prometheus.io/port", name): fmt.Sprintf("%d", port),
	}
}

// LabelsK8sApp returns some labels with "app.kubernetes.io" prefix.
// NOTE: these labels are deprecated, and they are used for TiDB Operator v1 compatibility.
// If you are developing a new feature, please use labels with "pingcap.com" prefix instead.
func LabelsK8sApp(cluster, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/instance":   cluster,
		"app.kubernetes.io/managed-by": "tidb-operator",
		"app.kubernetes.io/name":       "tidb-cluster",
	}
}
