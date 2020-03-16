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

func (am *autoScalerManager) syncTiDB(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Spec.TiDB == nil {
		return nil
	}
	sts, err := am.stsLister.StatefulSets(tc.Namespace).Get(operatorUtils.GetStatefulSetName(tc, v1alpha1.TiDBMemberType))
	if err != nil {
		return err
	}
	if !checkAutoScalingPrerequisites(tc, sts, v1alpha1.TiDBMemberType) {
		return nil
	}
	currentReplicas := tc.Spec.TiDB.Replicas
	instances := filterTidbInstances(tc)
	targetReplicas, err := calculateTidbMetrics(tac, sts, instances)
	if err != nil {
		return err
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	if targetReplicas == tc.Spec.TiDB.Replicas {
		return nil
	}
	return syncTiDBAfterCalculated(tc, tac, currentReplicas, targetReplicas, sts)
}

// syncTiDBAfterCalculated would check the Consecutive count to avoid jitter, and it would also check the interval
// duration between each auto-scaling. If either of them is not meet, the auto-scaling would be rejected.
// If the auto-scaling is permitted, the timestamp would be recorded and the Consecutive count would be zeroed.
func syncTiDBAfterCalculated(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, currentReplicas, recommendedReplicas int32, sts *appsv1.StatefulSet) error {
	intervalSeconds := tac.Spec.TiDB.ScaleInIntervalSeconds
	if recommendedReplicas > currentReplicas {
		intervalSeconds = tac.Spec.TiDB.ScaleOutIntervalSeconds
	}
	ableToScale, err := checkStsAutoScalingInterval(tac, *intervalSeconds, v1alpha1.TiDBMemberType)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	return updateTcTiDBIfScale(tc, tac, recommendedReplicas)
}

// Currently we didnt' record the auto-scaling out slot for tidb, because it is pointless for now.
func updateTcTiDBIfScale(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, recommendedReplicas int32) error {
	tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Unix())
	tc.Spec.TiDB.Replicas = recommendedReplicas
	tac.Status.TiDBStatus.LastScaleTime = time.Now().Format(time.RFC3339)
	return nil
}

func calculateTidbMetrics(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet, instances []string) (int32, error) {
	ep, err := genMetricsEndpoint(tac)
	if err != nil {
		return -1, err
	}
	client, err := promClient.NewClient(promClient.Config{Address: ep})
	if err != nil {
		return -1, err
	}
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
		Endpoint:  ep,
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
