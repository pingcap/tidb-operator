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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	promClient "github.com/prometheus/client_golang/api"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
)

//TODO: create issue to explain how auto-scaling algorithm based by cpu metrics work
func CalculateCpuMetrics(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet,
	client promClient.Client, instances []string, metric autoscalingv2beta2.MetricSpec,
	queryPattern, timeWindow string, memberType v1alpha1.MemberType) (int32, error) {
	if metric.Resource == nil || metric.Resource.Target.AverageUtilization == nil {
		return 0, fmt.Errorf(InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
	}
	currentReplicas := len(instances)
	c, err := filterContainer(tac, sts, memberType.String())
	if err != nil {
		return 0, err
	}
	cpuRequestsRadio, err := extractCpuRequestsRadio(c)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	duration, err := time.ParseDuration(timeWindow)
	if err != nil {
		return 0, err
	}
	prvious := now.Truncate(duration)
	r := &Response{}
	err = queryMetricsFromPrometheus(tac, fmt.Sprintf(queryPattern, tac.Spec.Cluster.Name), client, now.Unix(), r)
	if err != nil {
		return 0, err
	}
	sum1, err := sumByInstanceFromResponse(instances, r)
	if err != nil {
		return 0, err
	}
	err = queryMetricsFromPrometheus(tac, fmt.Sprintf(queryPattern, tac.Spec.Cluster.Name), client, prvious.Unix(), r)
	if err != nil {
		return 0, err
	}
	sum2, err := sumByInstanceFromResponse(instances, r)
	if err != nil {
		return 0, err
	}
	if sum1-sum2 < 0 {
		return 0, fmt.Errorf(CpuSumMetricsErrorMsg, tac.Namespace, tac.Name, timeWindow)
	}
	cpuSecsTotal := sum1 - sum2
	durationSeconds := duration.Seconds()
	utilizationRadio := float64(*metric.Resource.Target.AverageUtilization) / 100.0
	expectedCpuSecsTotal := cpuRequestsRadio * durationSeconds * float64(currentReplicas) * utilizationRadio
	rc, err := calculate(cpuSecsTotal, expectedCpuSecsTotal, int32(currentReplicas))
	if err != nil {
		return 0, err
	}
	return rc, nil
}

func extractCpuRequestsRadio(c *corev1.Container) (float64, error) {
	if c.Resources.Requests.Cpu() == nil || c.Resources.Requests.Cpu().MilliValue() < 1 {
		return 0, fmt.Errorf("container[%s] cpu requests is empty", c.Name)
	}
	return float64(c.Resources.Requests.Cpu().MilliValue()) / 1000.0, nil
}
