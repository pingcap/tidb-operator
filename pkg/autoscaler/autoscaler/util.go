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

	"github.com/jonboulle/clockwork"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

var defaultMetricSpec = autoscalingv2beta2.MetricSpec{
	Type: autoscalingv2beta2.ResourceMetricSourceType,
	Resource: &autoscalingv2beta2.ResourceMetricSource{
		Name: corev1.ResourceCPU,
		Target: autoscalingv2beta2.MetricTarget{
			AverageUtilization: pointer.Int32Ptr(80),
		},
	},
}

// checkStsAutoScalingPrerequisites would check the sts status to ensure wouldn't happen during
// upgrading, scaling
func checkStsAutoScalingPrerequisites(set *appsv1.StatefulSet) bool {
	if operatorUtils.IsStatefulSetUpgrading(set) {
		return false
	}
	if operatorUtils.IsStatefulSetScaling(set) {
		return false
	}
	return true
}

// checkLastSyncingTimestamp reset TiKV phase if last auto scaling timestamp is longer than thresholdSec
func checkLastSyncingTimestamp(tac *v1alpha1.TidbClusterAutoScaler, thresholdSec time.Duration, clock clockwork.Clock) (bool, error) {
	if tac.Annotations == nil {
		tac.Annotations = map[string]string{}
	}

	lastAutoScalingTimestamp, existed := tac.Annotations[label.AnnLastSyncingTimestamp]
	if !existed {
		// NOTE: because record autoscaler sync timestamp happens after check auto scale,
		// label will not exist during first sync, return allow auto scale in this case.
		return true, nil
	}
	t, err := strconv.ParseInt(lastAutoScalingTimestamp, 10, 64)
	if err != nil {
		return false, err
	}
	// if there's no resync within thresholdSec, reset TiKV phase to Normal
	if clock.Now().After(time.Unix(t, 0).Add(thresholdSec)) {
		tac.Status.TiKV.Phase = v1alpha1.NormalAutoScalerPhase
		return false, nil
	}
	return true, nil
}

// checkStsReadyAutoScalingTimestamp would check whether there is enough time window after ready to scale
func checkStsReadyAutoScalingTimestamp(tac *v1alpha1.TidbClusterAutoScaler, thresholdSeconds int32, clock clockwork.Clock) (bool, error) {
	readyAutoScalingTimestamp, existed := tac.Annotations[label.AnnTiKVReadyToScaleTimestamp]

	if !existed {
		tac.Annotations[label.AnnTiKVReadyToScaleTimestamp] = fmt.Sprintf("%d", clock.Now().Unix())
		return false, nil
	}
	t, err := strconv.ParseInt(readyAutoScalingTimestamp, 10, 32)
	if err != nil {
		return false, err
	}
	readyAutoScalingSec := int32(clock.Now().Sub(time.Unix(t, 0)).Seconds())
	if thresholdSeconds > readyAutoScalingSec {
		return false, nil
	}
	return true, nil
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
		return false, err
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
	if memberType == v1alpha1.TiDBMemberType {
		if tc.Status.TiDB.Phase != v1alpha1.NormalPhase {
			return false
		}
	} else if memberType == v1alpha1.TiKVMemberType {
		if !tc.Status.TiKV.Synced {
			return false
		}
		if tc.Status.TiKV.Phase != v1alpha1.NormalPhase {
			return false
		}
	} else {
		// Unknown MemberType
		return false
	}
	return true
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
			if len(tac.Spec.TiKV.Metrics) == 0 {
				tac.Spec.TiKV.Metrics = append(tac.Spec.TiKV.Metrics, defaultMetricSpec)
			}
			if tac.Spec.TiKV.MetricsTimeDuration == nil {
				tac.Spec.TiKV.MetricsTimeDuration = pointer.StringPtr("3m")
			}
		}
		if tac.Spec.TiKV.ReadyToScaleThresholdSeconds == nil {
			tac.Spec.TiKV.ReadyToScaleThresholdSeconds = pointer.Int32Ptr(30)
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
			if len(tac.Spec.TiDB.Metrics) == 0 {
				tac.Spec.TiDB.Metrics = append(tac.Spec.TiDB.Metrics, defaultMetricSpec)
			}
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
