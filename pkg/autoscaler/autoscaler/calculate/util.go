// Copyright 2020 PingCAP, Inc.
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

package calculate

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
)

// currently, we only choose one metrics to be computed.
// If there exists several metrics, we tend to choose ResourceMetricSourceType metric
func FilterMetrics(metrics []v1alpha1.CustomMetric, name corev1.ResourceName) []v1alpha1.CustomMetric {
	var list []v1alpha1.CustomMetric
	for _, m := range metrics {
		if m.Type == autoscalingv2beta2.ResourceMetricSourceType && m.Resource != nil && m.Resource.Name == name {
			list = append(list, m)
			break
		}
	}
	return list
}

// Response is used to marshal the data queried from Prometheus
type Response struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric Metric        `json:"metric"`
	Value  []interface{} `json:"value"`
}

type Metric struct {
	Cluster             string `json:"cluster,omitempty"`
	Instance            string `json:"instance"`
	Job                 string `json:"job,omitempty"`
	KubernetesNamespace string `json:"kubernetes_namespace,omitempty"`
	KubernetesNode      string `json:"kubernetes_node,omitempty"`
	KubernetesPodIp     string `json:"kubernetes_pod_ip,omitempty"`
}
