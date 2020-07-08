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

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/calculate"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	promClient "github.com/prometheus/client_golang/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (am *autoScalerManager) syncTiKV(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Spec.TiKV == nil {
		return nil
	}
	if tac.Status.TiKV == nil {
		tac.Status.TiKV = &v1alpha1.TikvAutoScalerStatus{}
	}
	sts, err := am.stsLister.StatefulSets(tc.Namespace).Get(operatorUtils.GetStatefulSetName(tc, v1alpha1.TiKVMemberType))
	if err != nil {
		return err
	}
	if !checkAutoScalingPrerequisites(tc, sts, v1alpha1.TiKVMemberType) {
		return nil
	}
	instances := filterTiKVInstances(tc)
	return calculateTiKVMetrics(tac, tc, sts, instances)
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

func calculateTiKVMetrics(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster, sts *appsv1.StatefulSet, instances []string) error {
	ep, err := genMetricsEndpoint(tac)
	if err != nil {
		return err
	}
	client, err := promClient.NewClient(promClient.Config{Address: ep})
	if err != nil {
		return err
	}
	duration, err := time.ParseDuration(*tac.Spec.TiKV.MetricsTimeDuration)
	if err != nil {
		return err
	}

	// check CPU
	metrics := calculate.FilterMetrics(tac.Spec.TiKV.Metrics, corev1.ResourceCPU)
	if len(metrics) > 0 {
		sq := &calculate.SingleQuery{
			Endpoint:  ep,
			Timestamp: time.Now().Unix(),
			Instances: instances,
			Metric:    metrics[0],
			Quary:     fmt.Sprintf(calculate.TikvSumCpuMetricsPattern, tac.Spec.Cluster.Name, *tac.Spec.TiKV.MetricsTimeDuration),
		}
		return calculateTiKVCPUMetrics(tac, tc, sts, sq, client, duration)
	}
	return nil
}

func calculateTiKVCPUMetrics(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster, sts *appsv1.StatefulSet, sq *calculate.SingleQuery, client promClient.Client, duration time.Duration) error {

	targetReplicas, err := calculate.CalculateRecomendedReplicasByCpuCosts(tac, sq, sts, client, v1alpha1.TiKVMemberType, duration)
	if err != nil {
		return err
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	if targetReplicas == tc.Spec.TiKV.Replicas {
		return nil
	}
	currentReplicas := int32(len(sq.Instances))
	err = syncTiKVAfterCalculated(tc, tac, currentReplicas, targetReplicas)
	if err != nil {
		return err
	}
	return addAnnotationMarkIfScaleOutDueToCPUMetrics(tc, currentReplicas, targetReplicas, sts)
}

func checkTiKVAutoScaling(tac *v1alpha1.TidbClusterAutoScaler, intervalSeconds int32) (bool, error) {
	if tac.Annotations == nil {
		tac.Annotations = map[string]string{}
	}
	ableToScale, err := checkStsAutoScalingInterval(tac, intervalSeconds, v1alpha1.TiKVMemberType)
	if err != nil {
		return false, err
	}
	if !ableToScale {
		return false, nil
	}
	return true, nil
}

// syncTiKVAfterCalculated would check the Consecutive count to avoid jitter, and it would also check the interval
// duration between each auto-scaling. If either of them is not meet, the auto-scaling would be rejected.
// If the auto-scaling is permitted, the timestamp would be recorded and the Consecutive count would be zeroed.
// The currentReplicas of TiKV calculated in auto-scaling is the count of the StateUp TiKV instance, so we need to
// add the number of other state tikv instance replicas when we update the TidbCluster.Spec.TiKV.Replicas
func syncTiKVAfterCalculated(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, currentReplicas, recommendedReplicas int32) error {
	intervalSeconds := tac.Spec.TiKV.ScaleInIntervalSeconds
	if recommendedReplicas > currentReplicas {
		intervalSeconds = tac.Spec.TiKV.ScaleOutIntervalSeconds
	}
	ableToScale, err := checkTiKVAutoScaling(tac, *intervalSeconds)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	return updateTcTiKVIfScale(tc, tac, recommendedReplicas)
}

// we record the auto-scaling out slot for tikv, in order to add special hot labels when they are created
func updateTcTiKVIfScale(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, recommendedReplicas int32) error {
	tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Unix())
	tc.Spec.TiKV.Replicas = recommendedReplicas
	tac.Status.TiKV.RecommendedReplicas = recommendedReplicas
	return nil
}

// Add mark for the scale out tikv in annotations in cpu metric case
func addAnnotationMarkIfScaleOutDueToCPUMetrics(tc *v1alpha1.TidbCluster, currentReplicas, recommendedReplicas int32, sts *appsv1.StatefulSet) error {
	if recommendedReplicas > currentReplicas {
		newlyScaleOutOrdinalSets := helper.GetPodOrdinals(recommendedReplicas, sts).Difference(helper.GetPodOrdinals(currentReplicas, sts))
		if newlyScaleOutOrdinalSets.Len() > 0 {
			if tc.Annotations == nil {
				tc.Annotations = map[string]string{}
			}
			existed := operatorUtils.GetAutoScalingOutSlots(tc, v1alpha1.TiKVMemberType)
			v, err := operatorUtils.Encode(newlyScaleOutOrdinalSets.Union(existed).List())
			if err != nil {
				return err
			}
			tc.Annotations[label.AnnTiKVAutoScalingOutOrdinals] = v
		}
	}
	return nil
}
