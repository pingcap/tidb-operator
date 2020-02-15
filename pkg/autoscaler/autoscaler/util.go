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
	promClient "github.com/prometheus/client_golang/api"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	annScaleOutSuffix = "tidb.pingcap.com/consecutive-scale-out-count"
	annScaleInSuffix  = "tidb.pingcap.com/consecutive-scale-in-count"

	invalidMemberTypeErrorMsg    = "tac[%s/%s] invalid set MemberType:%s"
	invalidTacAnnotationErrorMsg = "tac[%s/%s]'s tc[%s/%s] annotation invalid set,err:%v"
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

// checkStsAutoScalingInterval would check whether there is enough interval duration between every two auto-scaling
func checkStsAutoScalingInterval(tc *v1alpha1.TidbCluster, intervalSeconds int32, memberType v1alpha1.MemberType) (bool, error) {
	if tc.Annotations == nil {
		tc.Annotations = map[string]string{}
	}
	lastAutoScalingTimestamp, existed := tc.Annotations[label.AnnTiDBLastAutoScalingTimestamp]
	if memberType == v1alpha1.TiKVMemberType {
		lastAutoScalingTimestamp, existed = tc.Annotations[label.AnnTiKVLastAutoScalingTimestamp]
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
	if tac.Spec.TiKV != nil {
		if tac.Spec.TiKV.MinReplicas == nil {
			tac.Spec.TiKV.MinReplicas = pointer.Int32Ptr(1)
		}
		if len(tac.Spec.TiKV.Metrics) == 0 {
			tac.Spec.TiKV.Metrics = append(tac.Spec.TiKV.Metrics, defaultMetricSpec)
		}
		if tac.Spec.TiKV.ScaleInThreshold == nil {
			tac.Spec.TiKV.ScaleInThreshold = pointer.Int32Ptr(1)
		}
		if tac.Spec.TiKV.ScaleOutThreshold == nil {
			tac.Spec.TiKV.ScaleOutThreshold = pointer.Int32Ptr(1)
		}
		if tac.Spec.TiKV.MetricsTimeWindowSeconds == nil {
			tac.Spec.TiKV.MetricsTimeWindowSeconds = pointer.Int32Ptr(180)
		}
		if tac.Spec.TiKV.ScaleOutIntervalSeconds == nil {
			tac.Spec.TiKV.ScaleOutIntervalSeconds = pointer.Int32Ptr(300)
		}
		if tac.Spec.TiKV.ScaleInIntervalSeconds == nil {
			tac.Spec.TiKV.ScaleInIntervalSeconds = pointer.Int32Ptr(300)
		}
	}

	if tac.Spec.TiDB != nil {
		if tac.Spec.TiDB.MinReplicas == nil {
			tac.Spec.TiDB.MinReplicas = pointer.Int32Ptr(1)
		}
		if len(tac.Spec.TiDB.Metrics) == 0 {
			tac.Spec.TiDB.Metrics = append(tac.Spec.TiDB.Metrics, defaultMetricSpec)
		}
		if tac.Spec.TiDB.ScaleInThreshold == nil {
			tac.Spec.TiDB.ScaleInThreshold = pointer.Int32Ptr(1)
		}
		if tac.Spec.TiDB.ScaleOutThreshold == nil {
			tac.Spec.TiDB.ScaleOutThreshold = pointer.Int32Ptr(1)
		}
		if tac.Spec.TiDB.MetricsTimeWindowSeconds == nil {
			tac.Spec.TiDB.MetricsTimeWindowSeconds = pointer.Int32Ptr(180)
		}
		if tac.Spec.TiDB.ScaleOutIntervalSeconds == nil {
			tac.Spec.TiDB.ScaleOutIntervalSeconds = pointer.Int32Ptr(300)
		}
		if tac.Spec.TiDB.ScaleInIntervalSeconds == nil {
			tac.Spec.TiDB.ScaleInIntervalSeconds = pointer.Int32Ptr(300)
		}
	}
}

// updateConsecutiveCount would update the tc annotation depended by the given replicas in each reconciling
func updateConsecutiveCount(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler,
	memberType v1alpha1.MemberType, currentReplicas int32, recommendedReplicas int32) error {
	if tc.Annotations == nil {
		tc.Annotations = map[string]string{}
	}

	targetScaleOutAnn := fmt.Sprintf("%s.%s", memberType.String(), annScaleOutSuffix)
	targetScaleInAnn := fmt.Sprintf("%s.%s", memberType.String(), annScaleInSuffix)

	var scaleOutCount int
	var scaleInCount int
	scaleOutCount, scaleInCount = 0, 0
	var err error

	if v, existed := tc.Annotations[targetScaleOutAnn]; existed {
		scaleOutCount, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf(invalidTacAnnotationErrorMsg, tac.Namespace, tac.Name, tc.Namespace, tc.Name, err)
		}
	}

	if v, existed := tc.Annotations[targetScaleInAnn]; existed {
		scaleInCount, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf(invalidTacAnnotationErrorMsg, tac.Namespace, tac.Name, tc.Namespace, tc.Name, err)
		}
	}

	if currentReplicas < recommendedReplicas {
		// scale-out
		scaleOutCount = scaleOutCount + 1
		scaleInCount = 0
	} else if currentReplicas > recommendedReplicas {
		// scale-in
		scaleOutCount = 0
		scaleInCount = scaleInCount + 1
	} else {
		scaleOutCount = 0
		scaleInCount = 0
	}

	// update tc annotation
	tc.Annotations[targetScaleOutAnn] = fmt.Sprintf("%d", scaleOutCount)
	tc.Annotations[targetScaleInAnn] = fmt.Sprintf("%d", scaleInCount)
	return nil
}

func checkConsecutiveCount(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler,
	memberType v1alpha1.MemberType, currentReplicas int32, recommendedReplicas int32) (bool, error) {
	if currentReplicas == recommendedReplicas {
		return false, nil
	}
	targetScaleOutAnn := fmt.Sprintf("%s.%s", memberType.String(), annScaleOutSuffix)
	targetScaleInAnn := fmt.Sprintf("%s.%s", memberType.String(), annScaleInSuffix)
	currentScaleOutCount, err := strconv.ParseInt(tc.Annotations[targetScaleOutAnn], 10, 32)
	if err != nil {
		return false, err
	}
	currentScaleInCount, err := strconv.ParseInt(tc.Annotations[targetScaleInAnn], 10, 32)
	if err != nil {
		return false, err
	}
	switch memberType {
	case v1alpha1.TiDBMemberType:
		if currentReplicas < recommendedReplicas {
			// scale-out
			if int32(currentScaleOutCount) < *tac.Spec.TiDB.ScaleOutThreshold {
				return false, nil
			}
		} else {
			// scale-in, no-scaling would be return nil at first
			if int32(currentScaleInCount) < *tac.Spec.TiDB.ScaleInThreshold {
				return false, nil
			}
		}
	case v1alpha1.TiKVMemberType:
		if currentReplicas < recommendedReplicas {
			// scale-out
			if int32(currentScaleOutCount) < *tac.Spec.TiKV.ScaleOutThreshold {
				return false, nil
			}
		} else {
			// scale-in, no-scaling would be return nil at first
			if int32(currentScaleInCount) < *tac.Spec.TiDB.ScaleInThreshold {
				return false, nil
			}
		}
	default:
		return false, fmt.Errorf(invalidMemberTypeErrorMsg, tac.Namespace, tac.Name, memberType)
	}
	return true, nil
}

func emptyConsecutiveCount(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) {
	targetScaleOutAnn := fmt.Sprintf("%s.%s", memberType.String(), annScaleOutSuffix)
	targetScaleInAnn := fmt.Sprintf("%s.%s", memberType.String(), annScaleInSuffix)
	tc.Annotations[targetScaleOutAnn] = "0"
	tc.Annotations[targetScaleInAnn] = "0"
}

//TODO: calculate the recommended replicas from Prometheus
func calculateRecommendedReplicas(tac *v1alpha1.TidbClusterAutoScaler, memberType v1alpha1.MemberType,
	client promClient.Client) int32 {
	return 0
}
