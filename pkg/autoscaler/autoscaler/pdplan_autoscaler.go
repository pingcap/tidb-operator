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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const groupLabelKey = "group"

func (am *autoScalerManager) syncPlans(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, plans []pdapi.Plan) error {
	groupNames := sets.String{}
	groupPlanMap := make(map[string]pdapi.Plan)
	for _, plan := range plans {
		groupName := plan.Labels[groupLabelKey]
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

	toUpdate := groupNames.Intersection(existedGroups)
	err = am.updateAutoscalingClusters(tac, toUpdate.UnsortedList(), groupTcMap, groupPlanMap)
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
			newTm := monitor.DeepCopy()
			clusters := make([]v1alpha1.TidbClusterRef, 0, len(newTm.Spec.Clusters)-1)
			for _, cluster := range newTm.Spec.Clusters {
				if cluster.Name != deleteTc.Name {
					clusters = append(clusters, cluster)
				}
			}
			newTm.Spec.Clusters = clusters
			am.updateTidbMonitor(newTm, func(oldClusters []v1alpha1.TidbClusterRef) []v1alpha1.TidbClusterRef {
				clusters := make([]v1alpha1.TidbClusterRef, 0, len(oldClusters)-1)
				for _, cluster := range oldClusters {
					if cluster.Name != deleteTc.Name {
						clusters = append(clusters, cluster)
					}
				}
				return clusters
			})
		}
	}
	return nil
}

func (am *autoScalerManager) updateAutoscalingClusters(tac *v1alpha1.TidbClusterAutoScaler, groups []string, groupTcMap map[string]*v1alpha1.TidbCluster, groupPlanMap map[string]pdapi.Plan) error {
	for _, group := range groups {
		actual, oldTc, expected := groupTcMap[group].DeepCopy(), groupTcMap[group], groupPlanMap[group]
		component := expected.Component

		if !checkAutoscalingComponent(tac, component) {
			continue
		}

		switch component {
		case "tikv":
			sts, err := am.stsLister.StatefulSets(actual.Namespace).Get(operatorUtils.GetStatefulSetName(actual, v1alpha1.TiKVMemberType))
			if err != nil {
				return err
			}
			if !checkAutoScalingPrerequisites(actual, sts, v1alpha1.TiKVMemberType) {
				continue
			}
			actual.Spec.TiKV.Replicas = int32(expected.Count)
		case "tidb":
			sts, err := am.stsLister.StatefulSets(actual.Namespace).Get(operatorUtils.GetStatefulSetName(actual, v1alpha1.TiDBMemberType))
			if err != nil {
				return err
			}
			if !checkAutoScalingPrerequisites(actual, sts, v1alpha1.TiDBMemberType) {
				continue
			}
			actual.Spec.TiDB.Replicas = int32(expected.Count)
		}

		_, err := am.tcControl.UpdateTidbCluster(actual, &actual.Status, &oldTc.Status)
		if err != nil {
			return err
		}
	}
	return nil
}

func (am *autoScalerManager) createAutoscalingClusters(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, groupsToCreate []string, groupPlanMap map[string]pdapi.Plan) error {
	for _, group := range groupsToCreate {
		plan := groupPlanMap[group]
		component := plan.Component

		if !checkAutoscalingComponent(tac, component) {
			continue
		}

		resource, ok := findAutoResource(tac.Spec.Resources, plan.ResourceType)
		if !ok {
			return fmt.Errorf("unknown resource type %v", plan.ResourceType)
		}
		resList := corev1.ResourceList{
			corev1.ResourceCPU:     resource.CPU,
			corev1.ResourceStorage: resource.Storage,
			corev1.ResourceMemory:  resource.Memory,
		}

		autoTcName, err := genAutoClusterName(tac, component, plan.Labels, resource)
		if err != nil {
			return err
		}
		autoTc := &v1alpha1.TidbCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      autoTcName,
				Namespace: tc.Namespace,
			},
			Spec: v1alpha1.TidbClusterSpec{
				Cluster: &v1alpha1.TidbClusterRef{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			},
		}

		switch component {
		case "tikv":
			autoTc.Spec.TiKV = autoTc.Spec.TiKV.DeepCopy()
			autoTc.Spec.TiKV.Replicas = int32(plan.Count)
			autoTc.Spec.TiKV.ResourceRequirements = corev1.ResourceRequirements{
				Limits:   resList,
				Requests: resList,
			}
			for k, v := range plan.Labels {
				autoTc.Spec.TiKV.Config.Server.Labels[k] = v
			}
		case "tidb":
			autoTc.Spec.TiDB = autoTc.Spec.TiDB.DeepCopy()
			autoTc.Spec.TiDB.ResourceRequirements = corev1.ResourceRequirements{
				Limits:   resList,
				Requests: resList,
			}
			for k, v := range plan.Labels {
				autoTc.Spec.TiDB.Config.Labels[k] = v
			}
		}

		// Patch custom labels
		autoTc.Labels[label.AutoInstanceLabelKey] = tac.Name
		autoTc.Labels[label.AutoComponentLabelKey] = component
		autoTc.Labels[label.AutoScalingGroupLabelKey] = group

		created, err := am.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(autoTc)
		if err != nil {
			klog.Errorf("cannot create new TidbCluster err:%v\n", err)
			return err
		}

		if monitorRef := tc.Status.Monitor; monitorRef != nil {
			monitor, err := am.tmLister.TidbMonitors(monitorRef.Namespace).Get(monitorRef.Name)
			if err != nil {
				return err
			}
			clusterRef := v1alpha1.TidbClusterRef{Name: created.Name, Namespace: created.Namespace}
			updatedTm := monitor.DeepCopy()
			updatedTm.Spec.Clusters = append(updatedTm.Spec.Clusters, clusterRef)
			am.updateTidbMonitor(updatedTm,
				func(clusters []v1alpha1.TidbClusterRef) []v1alpha1.TidbClusterRef {
					return append(clusters, clusterRef)
				})
		}
	}
	return nil
}

func (am *autoScalerManager) updateTidbMonitor(tm *v1alpha1.TidbMonitor, onConflict func([]v1alpha1.TidbClusterRef) []v1alpha1.TidbClusterRef) error {
	ns := tm.GetNamespace()
	tmName := tm.GetName()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		_, updateErr = am.cli.PingcapV1alpha1().TidbMonitors(ns).Update(tm)
		if updateErr == nil {
			klog.Infof("TidbMonitor: [%s/%s] updated successfully", ns, tmName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbMonitor: [%s/%s], error: %v", ns, tmName, updateErr)
		if updated, err := am.tmLister.TidbMonitors(ns).Get(tmName); err == nil {
			// make a copy so we don't mutate the shared cache
			tm = updated.DeepCopy()
			tm.Spec.Clusters = onConflict(tm.Spec.Clusters)
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbMonitor %s/%s from lister: %v", ns, tmName, err))
		}
		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update TidbMonitor: [%s/%s], error: %v", ns, tmName, err)
	}
	return err
}
