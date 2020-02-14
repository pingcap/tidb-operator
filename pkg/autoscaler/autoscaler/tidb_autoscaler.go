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
		return nil
	}
	sts, err := am.stsLister.StatefulSets(tc.Namespace).Get(operatorUtils.GetStatefulSetName(tc, v1alpha1.TiDBMemberType))
	if err != nil {
		return err
	}
	if !checkAutoScalingPrerequisites(tc, sts, v1alpha1.TiDBMemberType) {
		return nil
	}
	targetReplicas := tc.Spec.TiDB.Replicas

	// TODO: sync tidb.metrics from prometheus
	// rate(process_cpu_seconds_total{cluster="tidb",job="tidb"}[threshold Minute])
	//for _, _ = range tac.Spec.TiDB.Metrics {
	//	// revive:disable:empty-block
	//}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiDBMemberType)
	if targetReplicas == tc.Spec.TiDB.Replicas {
		emptyConsecutiveCount(tc, v1alpha1.TiDBMemberType)
		return nil
	}
	if err := updateConsecutiveCount(tc, tac, v1alpha1.TiDBMemberType, tc.Spec.TiDB.Replicas, targetReplicas); err != nil {
		return err
	}

	ableToScale, err := checkConsecutiveCount(tc, tac, v1alpha1.TiDBMemberType, tc.Spec.TiDB.Replicas, targetReplicas)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}

	intervalSeconds := tac.Spec.TiDB.ScaleInIntervalSeconds
	if targetReplicas > tc.Spec.TiDB.Replicas {
		intervalSeconds = tac.Spec.TiDB.ScaleOutIntervalSeconds
	}
	ableToScale, err = checkStsAutoScalingInterval(tc, *intervalSeconds, v1alpha1.TiDBMemberType)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	updateTcTiDBAnnIfScale(tc)
	tc.Spec.TiDB.Replicas = targetReplicas
	return nil
}

func updateTcTiDBAnnIfScale(tc *v1alpha1.TidbCluster) {
	tc.Spec.Annotations[label.AnnTiDBLastAutoScalingTimestamp] = time.Now().String()
	emptyConsecutiveCount(tc, v1alpha1.TiDBMemberType)
}
