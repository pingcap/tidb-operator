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
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
)

// MetricType describe the current Supported Metric Type to calculate the recommended Replicas
type MetricType string

const (
	MetricTypeCPU MetricType = "cpu"
	//metricTypeQPS MetricType = "qps"
)

// currently, we only choose one metrics to be computed.
// If there exists several metrics, we tend to choose ResourceMetricSourceType metric
func FilterMetrics(metrics []autoscalingv2beta2.MetricSpec) autoscalingv2beta2.MetricSpec {
	for _, m := range metrics {
		if m.Type == autoscalingv2beta2.ResourceMetricSourceType && m.Resource != nil {
			return m
		}
	}
	return metrics[0]
}

// genMetricType return the supported MetricType in Operator by kubernetes auto-scaling MetricType
func GenMetricType(tac *v1alpha1.TidbClusterAutoScaler, metric autoscalingv2beta2.MetricSpec) (MetricType, error) {
	if metric.Type == autoscalingv2beta2.ResourceMetricSourceType && metric.Resource != nil && metric.Resource.Name == corev1.ResourceCPU {
		return MetricTypeCPU, nil
	}
	return "", fmt.Errorf(InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
}

// filterContainer is to filter the specific container from the given statefulset(tidb/tikv)
func filterContainer(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet, containerName string) (*corev1.Container, error) {
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == containerName {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("tac[%s/%s]'s Target have not %s container", tac.Namespace, tac.Name, containerName)
}

func addMetricsStatusIntoMetricsStatusList(metrics v1alpha1.MetricsStatus, basicStatus *v1alpha1.BasicAutoScalerStatus) {
	if basicStatus.MetricsStatusList == nil {
		basicStatus.MetricsStatusList = []v1alpha1.MetricsStatus{}
	}
	for id, m := range basicStatus.MetricsStatusList {
		if m.Name == metrics.Name {
			basicStatus.MetricsStatusList[id] = metrics
			return
		}
	}
	basicStatus.MetricsStatusList = append(basicStatus.MetricsStatusList, metrics)
	return
}

const (
	statusSuccess = "success"
)

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
