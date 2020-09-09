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
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/query"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	promClient "github.com/prometheus/client_golang/api"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func (am *autoScalerManager) syncTiKVByPlans(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, tikvPlans []pdapi.Plan) error {
	if tac.Spec.TiKV == nil {
		return nil
	}
	if tac.Status.TiKV == nil {
		return nil
	}

	groupNames := sets.String{}
	groupPlanMap := make(map[string]pdapi.Plan)
	for _, plan := range tikvPlans {
		groupName := findAutoscalingGroupNameInLabels(plan.Labels)
		groupNames.Insert(groupName)
		groupPlanMap[groupName] = plan
	}
	requirement, err := labels.NewRequirement(label.AutoScalingGroupLabelKey, selection.In, groupNames.List())
	if err != nil {
		return err
	}
	selector := labels.NewSelector().Add(*requirement)

	tcList, err := am.tcLister.List(selector)
	if err != nil {
		return err
	}

	existedGroups := sets.String{}
	groupTcMap := make(map[string]*v1alpha1.TidbCluster)
	for _, tc := range tcList {
		groupName := tc.Labels[label.AutoScalingGroupLabelKey]
		existedGroups.Insert(groupName)
		groupTcMap[groupName] = tc
	}

	toDelete := existedGroups.Difference(groupNames)
	err = am.deleteAutoscalingClusters(tc, toDelete.UnsortedList(), groupTcMap)
	if err != nil {
		return err
	}

	toSync := groupNames.Intersection(existedGroups)
	err = am.updateAutoscalingClusters(toSync.UnsortedList(), groupTcMap, groupPlanMap)
	if err != nil {
		return err
	}

	toCreate := groupNames.Difference(existedGroups)
	err = am.createAutoscalingClusters(tc, tac, toCreate.UnsortedList(), groupPlanMap)
	if err != nil {
		return err
	}

	return nil
}

func (am *autoScalerManager) deleteAutoscalingClusters(tc *v1alpha1.TidbCluster, groupsToDelete []string, groupTcMap map[string]*v1alpha1.TidbCluster) error {
	for _, group := range groupsToDelete {
		deleteTc := groupTcMap[group]
		err := am.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Delete(deleteTc.Name, nil)
		if err != nil {
			return err
		}

		// Remove the cluster in the monitor
		if monitorRef := deleteTc.Status.Monitor; monitorRef != nil {
			monitor, err := am.tmLister.TidbMonitors(monitorRef.Namespace).Get(monitorRef.Name)
			if err != nil {
				return err
			}
			updated := monitor.DeepCopy()
			clusters := make([]v1alpha1.TidbClusterRef, 0, len(updated.Spec.Clusters)-1)
			for _, cluster := range updated.Spec.Clusters {
				if cluster.Name != deleteTc.Name {
					clusters = append(clusters, cluster)
				}
			}
			updated.Spec.Clusters = clusters
			_, err = am.cli.PingcapV1alpha1().TidbMonitors(monitor.Namespace).Update(updated)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (am *autoScalerManager) updateAutoscalingClusters(groups []string, groupTcMap map[string]*v1alpha1.TidbCluster, groupPlanMap map[string]pdapi.Plan) error {
	for _, group := range groups {
		actual, expected := groupTcMap[group].DeepCopy(), groupPlanMap[group]
		actual.Spec.TiKV.Replicas = int32(expected.Count)
		_, err := am.cli.PingcapV1alpha1().TidbClusters(actual.Namespace).Update(actual)
		if err != nil {
			return err
		}
	}
	return nil
}

func (am *autoScalerManager) createAutoscalingClusters(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, groupsToCreate []string, groupPlanMap map[string]pdapi.Plan) error {
	for _, group := range groupsToCreate {
		plan := groupPlanMap[group]
		var resource v1alpha1.AutoResource
		for _, res := range tac.Spec.Resources {
			if res.ResourceType == plan.ResourceType {
				resource = res
				break
			}
		}
		resList := corev1.ResourceList{
			corev1.ResourceCPU:     resource.CPU,
			corev1.ResourceStorage: resource.Storage,
			corev1.ResourceMemory:  resource.Memory,
		}
		tc := &v1alpha1.TidbCluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      group,
				Namespace: tc.Namespace,
			},
			Spec: v1alpha1.TidbClusterSpec{
				TiKV: &v1alpha1.TiKVSpec{
					Replicas: int32(plan.Count),
					ResourceRequirements: corev1.ResourceRequirements{
						Limits:   resList,
						Requests: resList,
					},
				},
				Cluster: &v1alpha1.TidbClusterRef{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			},
		}

		created, err := am.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
		if err != nil {
			klog.Errorf("cannot create new TidbCluster %v\n", err)
			return err
		}

		if monitorRef := tc.Status.Monitor; monitorRef != nil {
			monitor, err := am.tmLister.TidbMonitors(monitorRef.Namespace).Get(monitorRef.Name)
			if err != nil {
				return err
			}
			updated := monitor.DeepCopy()
			updated.Spec.Clusters = append(updated.Spec.Clusters, v1alpha1.TidbClusterRef{Name: created.Name, Namespace: created.Namespace})
			am.cli.PingcapV1alpha1().TidbMonitors(updated.Namespace).Update(updated)
		}
	}
	return nil
}

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
	return calculateTiKVMetrics(tac, tc, sts, instances, am.kubecli)
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

func calculateTiKVMetrics(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster, sts *appsv1.StatefulSet, instances []string, kubecli kubernetes.Interface) error {
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
	if len(tac.Spec.TiKV.Metrics) < 1 {
		klog.V(4).Infof("tac[%s/%s] have no setting, skip auto-scaling", tac.Namespace, tac.Name)
		return nil
	}

	// check externalConfig
	if tac.Spec.TiKV.External != nil {
		return calculateTiKVExternalService(tc, tac, sts, kubecli)
	}

	// check CPU
	metrics := calculate.FilterMetrics(tac.Spec.TiKV.Metrics, corev1.ResourceCPU)
	if len(metrics) > 0 {
		sq := &calculate.SingleQuery{
			Endpoint:  ep,
			Timestamp: time.Now().Unix(),
			Instances: instances,
			Query:     fmt.Sprintf(calculate.TikvSumCpuMetricsPattern, tac.Spec.Cluster.Name, *tac.Spec.TiKV.MetricsTimeDuration),
		}
		return calculateTiKVCPUMetrics(tac, tc, sts, sq, client, duration, metrics[0])
	}

	// check storage
	metrics = calculate.FilterMetrics(tac.Spec.TiKV.Metrics, corev1.ResourceStorage)
	if len(metrics) > 0 {
		now := time.Now().Unix()
		capacitySq := &calculate.SingleQuery{
			Endpoint:  ep,
			Timestamp: now,
			Instances: instances,
			Query:     fmt.Sprintf(calculate.TikvSumStorageMetricsPattern, tac.Spec.Cluster.Name, "capacity"),
		}
		availableSq := &calculate.SingleQuery{
			Endpoint:  ep,
			Timestamp: now,
			Instances: instances,
			Query:     fmt.Sprintf(calculate.TikvSumStorageMetricsPattern, tac.Spec.Cluster.Name, "available"),
		}
		return calculateTiKVStorageMetrics(tac, tc, capacitySq, availableSq, client, metrics[0])
	}

	// none metrics selected, end auto-scaling
	return nil
}

func calculateTiKVExternalService(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, sts *appsv1.StatefulSet, kubecli kubernetes.Interface) error {
	targetReplicas, err := query.ExternalService(tc, v1alpha1.TiKVMemberType, tac.Spec.TiKV.External.Endpoint, kubecli)
	if err != nil {
		klog.Errorf("tac[%s/%s] 's query to the external endpoint got error: %v", tac.Namespace, tac.Name, err)
		return err
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	if targetReplicas == tc.Spec.TiKV.Replicas {
		return nil
	}
	currentReplicas := tc.Spec.TiKV.Replicas
	intervalSeconds := tac.Spec.TiKV.ScaleInIntervalSeconds
	if targetReplicas > currentReplicas {
		intervalSeconds = tac.Spec.TiKV.ScaleOutIntervalSeconds
	}
	ableToScale, err := checkTiKVAutoScalingInterval(tac, *intervalSeconds)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	err = updateTacIfTiKVScale(tc, tac, targetReplicas)
	if err != nil {
		return err
	}
	return addAnnotationMarkIfScaleOutDueToCPUMetrics(tc, currentReplicas, targetReplicas, sts)
}

func calculateTiKVStorageMetrics(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster,
	capSq, avaSq *calculate.SingleQuery, client promClient.Client, metric v1alpha1.CustomMetric) error {
	if tc.Spec.TiKV.Replicas >= tac.Spec.TiKV.MaxReplicas {
		klog.V(4).Infof("tac[%s/%s]'s tikv won't scale out by storage pressure due to maxReplicas", tac.Namespace, tac.Name)
		return nil
	}
	intervalSeconds := tac.Spec.TiKV.ScaleOutIntervalSeconds
	ableToScale, err := checkTiKVAutoScalingInterval(tac, *intervalSeconds)
	if err != nil {
		return err
	}
	if !ableToScale {
		klog.Infof("tac[%s/%s]'s tikv won't scale out by storage pressure due to scale-out cool-down interval", tac.Namespace, tac.Name)
		return nil
	}
	storagePressure, err := calculate.CalculateWhetherStoragePressure(tac, capSq, avaSq, client, metric)
	if err != nil {
		return err
	}
	if !storagePressure {
		return nil
	}
	ableToScale, err = checkWhetherAbleToScaleDueToStorage(tac, metric, time.Now(), controller.ResyncDuration)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	currentReplicas := tc.Spec.TiKV.Replicas
	targetReplicas := currentReplicas + 1
	return updateTacIfTiKVScale(tc, tac, targetReplicas)
}

func calculateTiKVCPUMetrics(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster, sts *appsv1.StatefulSet,
	sq *calculate.SingleQuery, client promClient.Client, duration time.Duration, metric v1alpha1.CustomMetric) error {

	targetReplicas, err := calculate.CalculateRecomendedReplicasByCpuCosts(tac, sq, sts, client, v1alpha1.TiKVMemberType, duration, metric.MetricSpec)
	if err != nil {
		return err
	}
	targetReplicas = limitTargetReplicas(targetReplicas, tac, v1alpha1.TiKVMemberType)
	if targetReplicas == tc.Spec.TiKV.Replicas {
		return nil
	}
	currentReplicas := int32(len(sq.Instances))
	intervalSeconds := tac.Spec.TiKV.ScaleInIntervalSeconds
	if targetReplicas > currentReplicas {
		intervalSeconds = tac.Spec.TiKV.ScaleOutIntervalSeconds
	}
	ableToScale, err := checkTiKVAutoScalingInterval(tac, *intervalSeconds)
	if err != nil {
		return err
	}
	if !ableToScale {
		return nil
	}
	err = updateTacIfTiKVScale(tc, tac, targetReplicas)
	if err != nil {
		return err
	}
	return addAnnotationMarkIfScaleOutDueToCPUMetrics(tc, currentReplicas, targetReplicas, sts)
}

// checkTiKVAutoScalingInterval check the each 2 auto-scaling interval depends on the scaling-in and scaling-out
// Note that for the storage scaling, we will check scale-out interval before we start to scraping metrics,
// and for the cpu scaling, we will check scale-in/scale-out interval after we finish calculating metrics.
func checkTiKVAutoScalingInterval(tac *v1alpha1.TidbClusterAutoScaler, intervalSeconds int32) (bool, error) {
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

// checkWhetherAbleToScaleDueToStorage will check whether the storage pressure status have been existed for as least
// LeastStoragePressurePeriodSeconds duration. If not, the operator would wait next round to check again.
func checkWhetherAbleToScaleDueToStorage(tac *v1alpha1.TidbClusterAutoScaler, metric v1alpha1.CustomMetric, now time.Time, resyncDuration time.Duration) (bool, error) {
	if metric.LeastStoragePressurePeriodSeconds == nil {
		return false, fmt.Errorf("tac[%s/%s]'s leastStoragePressurePeriodSeconds must be setted before scale out in storage", tac.Namespace, tac.Name)
	}
	if tac.Status.TiKV.LastAutoScalingTimestamp == nil {
		return false, fmt.Errorf("tac[%s/%s]'s tikv status LastAutoScalingTimestamp haven't been set", tac.Namespace, tac.Name)
	}
	if now.Sub(tac.Status.TiKV.LastAutoScalingTimestamp.Time) > 3*resyncDuration {
		klog.Infof("tac[%s/%s]'s tikv status LastAutoScalingTimestamp timeout", tac.Namespace, tac.Name)
		return false, nil
	}
	for _, m := range tac.Status.TiKV.MetricsStatusList {
		if m.Name == string(corev1.ResourceStorage) {
			if m.StoragePressure == nil || m.StoragePressureStartTime == nil {
				return false, nil
			}
			x := now.Sub(m.StoragePressureStartTime.Time).Seconds()
			if x >= float64(*metric.LeastStoragePressurePeriodSeconds) {
				return true, nil
			}
		}
	}
	return false, nil
}

// updateTacIfTiKVScale update the tac status and syncing annotations if tikv scale-in/out
func updateTacIfTiKVScale(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, recommendedReplicas int32) error {
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
