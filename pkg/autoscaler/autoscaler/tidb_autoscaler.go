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
	targetReplicas := calculateRecommendedReplicas(tac, v1alpha1.TiDBMemberType, client)
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
