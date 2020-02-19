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

package autoscaler

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/calculate"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	promClient "github.com/prometheus/client_golang/api"
	appsv1 "k8s.io/api/apps/v1"
)

func (am *autoScalerManager) syncTiDB(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, client promClient.Client) error {
	if tac.Spec.TiDB == nil {
		emptyAutoScalingCountAnn(tac, v1alpha1.TiDBMemberType)
		return nil
	}
	sts, err := am.stsLister.StatefulSets(tc.Namespace).Get(operatorUtils.GetStatefulSetName(tc, v1alpha1.TiDBMemberType))
	if err != nil {
		return err
	}
	if !checkAutoScalingPrerequisites(tc, sts, v1alpha1.TiDBMemberType) {
		emptyAutoScalingCountAnn(tac, v1alpha1.TiDBMemberType)
		return nil
	}
	currentReplicas := tc.Spec.TiDB.Replicas
	instances := filterTidbInstances(tc)
	targetReplicas, err := calculateTidbMetrics(tac, sts, client, instances)
	if err != nil {
		return err
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	if targetReplicas == tc.Spec.TiDB.Replicas {
		emptyAutoScalingCountAnn(tac, v1alpha1.TiDBMemberType)
		return nil
	}
	return syncTiDBAfterCalculated(tc, tac, currentReplicas, targetReplicas)
}

// syncTiDBAfterCalculated would check the Consecutive count to avoid jitter, and it would also check the interval
// duration between each auto-scaling. If either of them is not meet, the auto-scaling would be rejected.
// If the auto-scaling is permitted, the timestamp would be recorded and the Consecutive count would be zeroed.
func syncTiDBAfterCalculated(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, currentReplicas, recommendedReplicas int32) error {
	if err := updateConsecutiveCount(tac, v1alpha1.TiDBMemberType, currentReplicas, recommendedReplicas); err != nil {
		return err
	}

	ableToScale, err := checkConsecutiveCount(tac, v1alpha1.TiDBMemberType, currentReplicas, recommendedReplicas)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	intervalSeconds := tac.Spec.TiDB.ScaleInIntervalSeconds
	if recommendedReplicas > currentReplicas {
		intervalSeconds = tac.Spec.TiDB.ScaleOutIntervalSeconds
	}
	ableToScale, err = checkStsAutoScalingInterval(tc, *intervalSeconds, v1alpha1.TiDBMemberType)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	updateTcTiDBAnnIfScale(tac)
	tc.Spec.TiDB.Replicas = recommendedReplicas
	return nil
}

func updateTcTiDBAnnIfScale(tac *v1alpha1.TidbClusterAutoScaler) {
	tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp] = time.Now().String()
	emptyAutoScalingCountAnn(tac, v1alpha1.TiDBMemberType)
}

func calculateTidbMetrics(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet, client promClient.Client, instances []string) (int32, error) {
	metric := calculate.FilterMetrics(tac.Spec.TiDB.Metrics)
	mType, err := calculate.GenMetricType(tac, metric)
	if err != nil {
		return -1, err
	}
	duration, err := time.ParseDuration(*tac.Spec.TiDB.MetricsTimeDuration)
	if err != nil {
		return -1, err
	}
	sq := &calculate.SingleQuery{
		Timestamp: time.Now().Unix(),
		Instances: instances,
		Metric:    metric,
		Quary:     fmt.Sprintf(calculate.TidbSumCpuMetricsPattern, tac.Spec.Cluster.Name, *tac.Spec.TiDB.MetricsTimeDuration),
	}

	switch mType {
	case calculate.MetricTypeCPU:
		return calculate.CalculateRecomendedReplicasByCpuCosts(tac, sq, sts, client, v1alpha1.TiDBMemberType, duration)
	default:
		return -1, fmt.Errorf(calculate.InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
	}
}

func filterTidbInstances(tc *v1alpha1.TidbCluster) []string {
	var instances []string
	for i := 0; int32(i) < tc.Status.TiDB.StatefulSet.Replicas; i++ {
		podName := operatorUtils.GetPodName(tc, v1alpha1.TiDBMemberType, int32(i))
		if _, existed := tc.Status.TiDB.FailureMembers[podName]; !existed {
			instances = append(instances, podName)
		}
	}
	return instances
}
