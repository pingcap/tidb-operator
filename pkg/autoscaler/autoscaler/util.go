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
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

// checkStsAutoScalingPrerequisites would check the sts status to ensure wouldn't happen during
// upgrading, scaling
func checkStsAutoScalingPrerequisites(set *appsv1.StatefulSet) bool {
	return !operatorUtils.IsStatefulSetUpgrading(set) && !operatorUtils.IsStatefulSetScaling(set)
}

// checkStsAutoScalingInterval would check whether there is enough interval duration between every two auto-scaling
func checkStsAutoScalingInterval(tac *v1alpha1.TidbClusterAutoScaler, intervalSeconds int32, memberType v1alpha1.MemberType) (bool, error) {
	lastAutoScalingTimestamp, existed := tac.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
	if memberType == v1alpha1.TiKVMemberType {
		lastAutoScalingTimestamp, existed = tac.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
	}
	if !existed {
		return true, nil
	}
	t, err := strconv.ParseInt(lastAutoScalingTimestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("tac[%s/%s] parse last auto-scaling timestamp failed,err:%v", tac.Namespace, tac.Name, err)
	}
	if intervalSeconds > int32(time.Now().Sub(time.Unix(t, 0)).Seconds()) {
		return false, nil
	}
	return true, nil
}

// checkAutoScalingPrerequisites would check the tidbcluster status to ensure the autoscaling would'n happen during
// upgrading, scaling and syncing
func checkAutoScalingPrerequisites(tc *v1alpha1.TidbCluster, sts *appsv1.StatefulSet, memberType v1alpha1.MemberType) bool {
	if !checkStsAutoScalingPrerequisites(sts) {
		return false
	}
	switch memberType {
	case v1alpha1.TiDBMemberType:
		return tc.Status.TiDB.Phase == v1alpha1.NormalPhase
	case v1alpha1.TiKVMemberType:
		return tc.Status.TiKV.Synced && tc.Status.TiKV.Phase == v1alpha1.NormalPhase
	default:
		// Unknown MemberType
		return false
	}
}

// limitTargetReplicas would limit the calculated target replicas to ensure the min/max Replicas
func limitTargetReplicas(targetReplicas int32, tac *v1alpha1.TidbClusterAutoScaler, memberType v1alpha1.MemberType) int32 {
	var min, max int32
	switch memberType {
	case v1alpha1.TiKVMemberType:
		min, max = *tac.Spec.TiKV.MinReplicas, tac.Spec.TiKV.MaxReplicas
	case v1alpha1.TiDBMemberType:
		min, max = *tac.Spec.TiDB.MinReplicas, tac.Spec.TiDB.MaxReplicas
	default:
		return targetReplicas
	}
	if targetReplicas > max {
		return max
	}
	if targetReplicas < min {
		return min
	}
	return targetReplicas
}

// If the minReplicas not set, the default value would be 1
// If the Metrics not set, the default metric will be set to 80% average CPU utilization.
// defaultTAC would default the omitted value
func defaultTAC(tac *v1alpha1.TidbClusterAutoScaler) {
	if tac.Annotations == nil {
		tac.Annotations = map[string]string{}
	}

	def := func(spec *v1alpha1.BasicAutoScalerSpec) {
		if spec.MinReplicas == nil {
			spec.MinReplicas = pointer.Int32Ptr(1)
		}
		if spec.ScaleOutIntervalSeconds == nil {
			spec.ScaleOutIntervalSeconds = pointer.Int32Ptr(300)
		}
		if spec.ScaleInIntervalSeconds == nil {
			spec.ScaleInIntervalSeconds = pointer.Int32Ptr(500)
		}
		// If ExternalEndpoint is not provided, we would set default metrics
		if spec.External == nil && spec.MetricsTimeDuration == nil {
			spec.MetricsTimeDuration = pointer.StringPtr("3m")
		}
	}

	if tidb := tac.Spec.TiDB; tidb != nil {
		def(&tidb.BasicAutoScalerSpec)
	}

	if tikv := tac.Spec.TiKV; tikv != nil {
		def(&tikv.BasicAutoScalerSpec)
		for id, m := range tikv.Metrics {
			if m.Resource == nil || m.Resource.Name != corev1.ResourceStorage {
				continue
			}
			if m.LeastStoragePressurePeriodSeconds == nil {
				m.LeastStoragePressurePeriodSeconds = pointer.Int64Ptr(300)
			}
			if m.LeastRemainAvailableStoragePercent == nil {
				m.LeastRemainAvailableStoragePercent = pointer.Int64Ptr(10)
			}
			tikv.Metrics[id] = m
		}
	}

	if monitor := tac.Spec.Monitor; monitor != nil && len(monitor.Namespace) < 1 {
		monitor.Namespace = tac.Namespace
	}
}

func genMetricsEndpoint(tac *v1alpha1.TidbClusterAutoScaler) (string, error) {
	if tac.Spec.MetricsUrl == nil && tac.Spec.Monitor == nil {
		return "", fmt.Errorf("tac[%s/%s] metrics url or monitor should be defined explicitly", tac.Namespace, tac.Name)
	}
	if tac.Spec.MetricsUrl != nil {
		return *tac.Spec.MetricsUrl, nil
	}
	return fmt.Sprintf("http://%s-prometheus.%s.svc:9090", tac.Spec.Monitor.Name, tac.Spec.Monitor.Namespace), nil
}
