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
	"k8s.io/utils/pointer"
)

const (
	CpuSumMetricsErrorMsg = "tac[%s/%s] cpu sum metrics error, can't calculate the past %s cpu metrics, may caused by prometheus restart while data persistence not enabled"
)

//TODO: create issue to explain how auto-scaling algorithm based on cpu metrics work
func CalculateRecomendedReplicasByCpuCosts(tac *v1alpha1.TidbClusterAutoScaler, sq *SingleQuery, sts *appsv1.StatefulSet,
	client promClient.Client, memberType v1alpha1.MemberType, duration time.Duration, metric autoscalingv2beta2.MetricSpec) (int32, error) {
	instances := sq.Instances
	if metric.Resource == nil || metric.Resource.Target.AverageUtilization == nil {
		return -1, fmt.Errorf(InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
	}
	currentReplicas := len(instances)
	c, err := filterContainer(tac, sts, memberType.String())
	if err != nil {
		return -1, err
	}
	cpuRequestsRatio, err := extractCpuRequestsRatio(c)
	if err != nil {
		return -1, err
	}
	r := &Response{}
	err = queryMetricsFromPrometheus(tac, client, sq, r)
	if err != nil {
		return -1, err
	}
	sum, err := sumForEachInstance(instances, r)
	if err != nil {
		return -1, err
	}
	if sum < 0 {
		return -1, fmt.Errorf(CpuSumMetricsErrorMsg, tac.Namespace, tac.Name, duration.String())
	}
	cpuSecsTotal := sum
	durationSeconds := duration.Seconds()
	utilizationRatio := float64(*metric.Resource.Target.AverageUtilization) / 100.0
	expectedCpuSecsTotal := cpuRequestsRatio * durationSeconds * float64(currentReplicas) * utilizationRatio
	rc, err := calculate(cpuSecsTotal, expectedCpuSecsTotal, int32(currentReplicas))
	if err != nil {
		return -1, err
	}
	metrics := v1alpha1.MetricsStatus{
		Name:           string(corev1.ResourceCPU),
		CurrentValue:   pointer.StringPtr(fmt.Sprintf("%v", cpuSecsTotal)),
		ThresholdValue: pointer.StringPtr(fmt.Sprintf("%v", expectedCpuSecsTotal)),
	}
	if memberType == v1alpha1.TiKVMemberType {
		addMetricsStatusIntoMetricsStatusList(metrics, &tac.Status.TiKV.BasicAutoScalerStatus)
	} else if memberType == v1alpha1.TiDBMemberType {
		addMetricsStatusIntoMetricsStatusList(metrics, &tac.Status.TiDB.BasicAutoScalerStatus)
	}
	return rc, nil
}

func extractCpuRequestsRatio(c *corev1.Container) (float64, error) {
	if c.Resources.Requests.Cpu() == nil || c.Resources.Requests.Cpu().MilliValue() < 1 {
		return 0, fmt.Errorf("container[%s] cpu requests is empty", c.Name)
	}
	return float64(c.Resources.Requests.Cpu().MilliValue()) / 1000.0, nil
}
