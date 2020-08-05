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
	if memberType != v1alpha1.TiKVMemberType && memberType != v1alpha1.TiDBMemberType {
		return targetReplicas
	}
	if memberType == v1alpha1.TiKVMemberType {
		if targetReplicas > tac.Spec.TiKV.MaxReplicas {
			targetReplicas = tac.Spec.TiKV.MaxReplicas
		} else if targetReplicas < *tac.Spec.TiKV.MinReplicas {
			targetReplicas = *tac.Spec.TiKV.MinReplicas
		}
	} else if memberType == v1alpha1.TiDBMemberType {
		if targetReplicas > tac.Spec.TiDB.MaxReplicas {
			targetReplicas = tac.Spec.TiDB.MaxReplicas
		} else if targetReplicas < *tac.Spec.TiDB.MinReplicas {
			targetReplicas = *tac.Spec.TiDB.MinReplicas
		}
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
	if tac.Spec.TiKV != nil {
		if tac.Spec.TiKV.MinReplicas == nil {
			tac.Spec.TiKV.MinReplicas = pointer.Int32Ptr(1)
		}
		if tac.Spec.TiKV.ScaleOutIntervalSeconds == nil {
			tac.Spec.TiKV.ScaleOutIntervalSeconds = pointer.Int32Ptr(300)
		}
		if tac.Spec.TiKV.ScaleInIntervalSeconds == nil {
			tac.Spec.TiKV.ScaleInIntervalSeconds = pointer.Int32Ptr(500)
		}
		// If ExternalEndpoint is not provided, we would set default metrics
		if tac.Spec.TiKV.ExternalEndpoint == nil {
			if tac.Spec.TiKV.MetricsTimeDuration == nil {
				tac.Spec.TiKV.MetricsTimeDuration = pointer.StringPtr("3m")
			}
		}
		for id, m := range tac.Spec.TiKV.Metrics {
			if m.Resource != nil && m.Resource.Name == corev1.ResourceStorage {
				if m.LeastStoragePressurePeriodSeconds == nil {
					m.LeastStoragePressurePeriodSeconds = pointer.Int64Ptr(300)
				}
				if m.LeastRemainAvailableStoragePercent == nil {
					m.LeastRemainAvailableStoragePercent = pointer.Int64Ptr(10)
				}
				tac.Spec.TiKV.Metrics[id] = m
			}
		}
	}

	if tac.Spec.TiDB != nil {
		if tac.Spec.TiDB.MinReplicas == nil {
			tac.Spec.TiDB.MinReplicas = pointer.Int32Ptr(1)
		}
		if tac.Spec.TiDB.ScaleOutIntervalSeconds == nil {
			tac.Spec.TiDB.ScaleOutIntervalSeconds = pointer.Int32Ptr(300)
		}
		if tac.Spec.TiDB.ScaleInIntervalSeconds == nil {
			tac.Spec.TiDB.ScaleInIntervalSeconds = pointer.Int32Ptr(500)
		}
		if tac.Spec.TiDB.ExternalEndpoint == nil {
			if tac.Spec.TiDB.MetricsTimeDuration == nil {
				tac.Spec.TiDB.MetricsTimeDuration = pointer.StringPtr("3m")
			}
		}
	}

	if tac.Spec.Monitor != nil {
		if len(tac.Spec.Monitor.Namespace) < 1 {
			tac.Spec.Monitor.Namespace = tac.Namespace
		}
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
