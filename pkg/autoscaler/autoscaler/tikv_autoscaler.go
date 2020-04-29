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

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/calculate"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/query"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	promClient "github.com/prometheus/client_golang/api"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/klog"
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
	var targetReplicas int32
	if tac.Spec.TiDB.ExternalEndpoint == nil {
		targetReplicas, err = calculateTikvMetrics(tac, sts, instances)
		if err != nil {
			return err
		}
	} else {
		targetReplicas, err = query.ExternalService(tc, v1alpha1.TiKVMemberType, tac.Spec.TiKV.ExternalEndpoint, am.kubecli)
		if err != nil {
			klog.Errorf("tac[%s/%s] 's query to the external endpoint got error: %v", tac.Namespace, tac.Name, err)
			return err
		}
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	if targetReplicas == tc.Spec.TiKV.Replicas {
		return nil
	}
	currentReplicas := int32(len(instances))
	return syncTiKVAfterCalculated(tc, tac, currentReplicas, targetReplicas, sts)
}

// syncTiKVAfterCalculated would check the Consecutive count to avoid jitter, and it would also check the interval
// duration between each auto-scaling. If either of them is not meet, the auto-scaling would be rejected.
// If the auto-scaling is permitted, the timestamp would be recorded and the Consecutive count would be zeroed.
// The currentReplicas of TiKV calculated in auto-scaling is the count of the StateUp TiKV instance, so we need to
// add the number of other state tikv instance replicas when we update the TidbCluster.Spec.TiKV.Replicas
func syncTiKVAfterCalculated(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, currentReplicas, recommendedReplicas int32, sts *appsv1.StatefulSet) error {

	intervalSeconds := tac.Spec.TiKV.ScaleInIntervalSeconds
	if recommendedReplicas > tc.Spec.TiKV.Replicas {
		intervalSeconds = tac.Spec.TiKV.ScaleOutIntervalSeconds
	}
	ableToScale, err := checkStsAutoScalingInterval(tac, *intervalSeconds, v1alpha1.TiKVMemberType)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	return updateTcTiKVIfScale(tc, tac, currentReplicas, recommendedReplicas, sts)
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

// we record the auto-scaling out slot for tikv, in order to add special hot labels when they are created
func updateTcTiKVIfScale(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, currentReplicas, recommendedReplicas int32, sts *appsv1.StatefulSet) error {
	tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = fmt.Sprintf("%d", time.Now().Unix())
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
	tc.Spec.TiKV.Replicas = recommendedReplicas
	tac.Status.TiKV.RecommendedReplicas = &recommendedReplicas
	return nil
}

func calculateTikvMetrics(tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet, instances []string) (int32, error) {
	ep, err := genMetricsEndpoint(tac)
	if err != nil {
		return -1, err
	}
	client, err := promClient.NewClient(promClient.Config{Address: ep})
	if err != nil {
		return -1, err
	}

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
		Endpoint:  ep,
		Timestamp: time.Now().Unix(),
		Instances: instances,
		Metric:    metric,
		Quary:     fmt.Sprintf(calculate.TikvSumCpuMetricsPattern, tac.Spec.Cluster.Name, *tac.Spec.TiKV.MetricsTimeDuration),
	}

	switch mType {
	case calculate.MetricTypeCPU:
		return calculate.CalculateRecomendedReplicasByCpuCosts(tac, sq, sts, client, v1alpha1.TiKVMemberType, duration)
	default:
		return -1, fmt.Errorf(calculate.InvalidTacMetricConfigureMsg, tac.Namespace, tac.Name)
	}
}
