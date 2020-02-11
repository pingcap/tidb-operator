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
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
)

//sum(rate(tikv_thread_cpu_seconds_total{cluster="tidb"}[1m])) by (instance)
//sum(rate(tikv_grpc_msg_duration_seconds_count{cluster="tidb", type!="kv_gc"}[1m])) by (instance)
func (am *autoScalerManager) syncTiKV(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, client promClient.Client) error {
	if tac.Spec.TiKV == nil {
		return nil
	}
	sts, err := am.stsLister.StatefulSets(tc.Namespace).Get(operatorUtils.GetStatefulSetName(tc, v1alpha1.TiKVMemberType))
	if err != nil {
		return err
	}
	if !checkTiKVAutoScalingPrerequisites(tc, sts) {
		return nil
	}
	targetReplicas := tc.Spec.TiKV.Replicas
	for _, metric := range tac.Spec.TiKV.Metrics {
		if metric.Type == autoscalingv2beta2.ResourceMetricSourceType {
			//TODO: auto-scaler only support CPU AverageUtilization metrics And QPS AverageValues
			// sum(rate(tikv_thread_cpu_seconds_total{cluster="tidb"}[1m])) by (instance)
			// sum(rate(tikv_grpc_msg_duration_seconds_count{cluster="tidb", type!="kv_gc"}[1m])) by (instance)
		}
	}
	if targetReplicas == tc.Spec.TiKV.Replicas {
		return nil
	}
	intervalSeconds := tac.Spec.TiKV.ScaleInIntervalSeconds
	if targetReplicas > tc.Spec.TiKV.Replicas {
		intervalSeconds = tac.Spec.TiKV.ScaleOutIntervalSeconds
	}
	ableToScale, err := checkStsAutoScalingInterval(tc, *intervalSeconds, v1alpha1.TiKVMemberType)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	tc.Spec.Annotations[label.AnnTiKVLastAutoScalingTimestamp] = time.Now().String()
	tc.Spec.TiDB.Replicas = targetReplicas
	return nil
}
