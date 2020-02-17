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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	promClient "github.com/prometheus/client_golang/api"
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
	currentReplicas := getStateUpReplicas(tc)
	targetReplicas := calculateRecommendedReplicas(tac, v1alpha1.TiKVMemberType, client)
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	if targetReplicas == tc.Spec.TiKV.Replicas {
		emptyAutoScalingCountAnn(tac, v1alpha1.TiKVMemberType)
		return nil
	}
	return syncTiKVAfterCalculated(tc, tac, currentReplicas, targetReplicas, tc.Spec.TiKV.Replicas-currentReplicas)
}

// syncTiDBAfterCalculated would check the Consecutive count to avoid jitter, and it would also check the interval
// duration between each auto-scaling. If either of them is not meet, the auto-scaling would be rejected.
// If the auto-scaling is permitted, the timestamp would be recorded and the Consecutive count would be zeroed.
// The currentReplicas of TiKV calculated in auto-scaling is the count of the StateUp TiKV instance, so we need to
// add the number of other state tikv instance replicas when we update the TidbCluster.Spec.TiKV.Replicas
func syncTiKVAfterCalculated(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, currentReplicas, recommendedReplicas, failureStoreCount int32) error {
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
	tc.Spec.TiKV.Replicas = recommendedReplicas + failureStoreCount
	return nil
}

func getStateUpReplicas(tc *v1alpha1.TidbCluster) int32 {
	count := 0
	for _, store := range tc.Status.TiKV.Stores {
		if store.State == v1alpha1.TiKVStateUp {
			count = count + 1
		}
	}
	return int32(count)
}

func updateTcTiKVAnnIfScale(tac *v1alpha1.TidbClusterAutoScaler) {
	tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = time.Now().String()
	emptyAutoScalingCountAnn(tac, v1alpha1.TiKVMemberType)
}
