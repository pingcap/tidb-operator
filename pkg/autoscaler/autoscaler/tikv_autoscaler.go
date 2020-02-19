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

func (am *autoScalerManager) syncTiKV(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, client promClient.Client) error {
	if tac.Spec.TiKV == nil {
		emptyAutoScalingCountAnn(tac, v1alpha1.TiKVMemberType)
		return nil
	}
	sts, err := am.stsLister.StatefulSets(tc.Namespace).Get(operatorUtils.GetStatefulSetName(tc, v1alpha1.TiKVMemberType))
	if err != nil {
		return err
	}
	if !checkAutoScalingPrerequisites(tc, sts, v1alpha1.TiKVMemberType) {
		emptyAutoScalingCountAnn(tac, v1alpha1.TiKVMemberType)
		return nil
	}
	instances := filterTiKVInstances(tc)
	currentReplicas := int32(len(instances))
	targetReplicas, err := calculateTikvMetrics(tac, sts, client, instances)
	if err != nil {
		return err
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	if targetReplicas == tc.Spec.TiKV.Replicas {
		emptyAutoScalingCountAnn(tac, v1alpha1.TiKVMemberType)
		return nil
	}
	return syncTiKVAfterCalculated(tc, tac, currentReplicas, targetReplicas)
}

// syncTiKVAfterCalculated would check the Consecutive count to avoid jitter, and it would also check the interval
// duration between each auto-scaling. If either of them is not meet, the auto-scaling would be rejected.
// If the auto-scaling is permitted, the timestamp would be recorded and the Consecutive count would be zeroed.
// The currentReplicas of TiKV calculated in auto-scaling is the count of the StateUp TiKV instance, so we need to
// add the number of other state tikv instance replicas when we update the TidbCluster.Spec.TiKV.Replicas
func syncTiKVAfterCalculated(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, currentReplicas, recommendedReplicas int32) error {
	if err := updateConsecutiveCount(tac, v1alpha1.TiKVMemberType, currentReplicas, recommendedReplicas); err != nil {
		return err
	}

	ableToScale, err := checkConsecutiveCount(tac, v1alpha1.TiKVMemberType, currentReplicas, recommendedReplicas)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}

	intervalSeconds := tac.Spec.TiKV.ScaleInIntervalSeconds
	if recommendedReplicas > tc.Spec.TiKV.Replicas {
		intervalSeconds = tac.Spec.TiKV.ScaleOutIntervalSeconds
	}
	ableToScale, err = checkStsAutoScalingInterval(tc, *intervalSeconds, v1alpha1.TiKVMemberType)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	updateTcTiKVAnnIfScale(tac)
	tc.Spec.TiKV.Replicas = recommendedReplicas
	return nil
}

//TODO: fetch tikv instances info from pdapi in future
func filterTiKVInstances(tc *v1alpha1.TidbCluster) []string {
	var instances []string
	for _, store := range tc.Status.TiKV.Stores {
		if store.State == v1alpha1.TiKVStateUp {
			instances = append(instances, store.PodName)
		}
	}
	return instances
}

func updateTcTiKVAnnIfScale(tac *v1alpha1.TidbClusterAutoScaler) {
	tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = time.Now().String()
	emptyAutoScalingCountAnn(tac, v1alpha1.TiKVMemberType)
}

func calculateTikvMetrics(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet, client promClient.Client, instances []string) (int32, error) {
	metric := calculate.FilterMetrics(tac.Spec.TiKV.Metrics)
	mType, err := calculate.GenMetricType(tac, metric)
	if err != nil {
		return -1, err
	}

	duration, err := time.ParseDuration(*tac.Spec.TiKV.MetricsTimeDuration)
	if err != nil {
		return -1, err
	}
	sq := &calculate.SingleQuery{
		Timestamp: time.Now().Unix(),
		Instances: instances,
		Metric:    metric,
		Quary:     fmt.Sprintf(calculate.TikvSumCpuMetricsPattern, tac.Spec.Cluster.Name, duration.String()),
	}

	switch mType {
	case calculate.MetricTypeCPU:
		return calculate.CalculateCpuCosts(tac, sq, sts, client, v1alpha1.TiKVMemberType, duration)
	default:
		return -1, fmt.Errorf(calculate.InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
	}
}
