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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const groupLabelKey = "group"

func (am *autoScalerManager) getAutoScaledClusters(tac *v1alpha1.TidbClusterAutoScaler, components []v1alpha1.MemberType) (tcList []*v1alpha1.TidbCluster, err error) {
	componentStrings := make([]string, len(components))
	for _, component := range components {
		componentStrings = append(componentStrings, component.String())
	}

	// Filter all autoscaled TidbClusters
	requirement, err := labels.NewRequirement(label.AutoInstanceLabelKey, selection.Equals, []string{tac.Name})
	if err != nil {
		return
	}

	componentRequirement, err := labels.NewRequirement(label.AutoComponentLabelKey, selection.In, componentStrings)
	if err != nil {
		return
	}

	selector := labels.NewSelector().Add(*requirement).Add(*componentRequirement)
	tcList, err = am.tcLister.TidbClusters(tac.Spec.Cluster.Namespace).List(selector)
	return
}

func (am *autoScalerManager) syncPlans(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, plans []pdapi.Plan, component v1alpha1.MemberType) error {
	planGroups := sets.String{}
	groupPlanMap := make(map[string]pdapi.Plan)
	for _, plan := range plans {
		groupName := plan.Labels[groupLabelKey]
		planGroups.Insert(groupName)
		groupPlanMap[groupName] = plan
	}

	tcList, err := am.getAutoScaledClusters(tac, []v1alpha1.MemberType{component})
	if err != nil {
		return err
	}

	existedGroups := sets.String{}
	groupTcMap := make(map[string]*v1alpha1.TidbCluster)
	for _, tc := range tcList {
		groupName, ok := tc.Labels[label.AutoScalingGroupLabelKey]
		if !ok {
			// External Autoscaling Clusters do not have group
			continue
		}
		if len(groupName) == 0 {
			klog.Errorf("unexpected: tidbcluster [%s/%s] has empty value for label %s", tc.Namespace, tc.Name, label.AutoScalingGroupLabelKey)
			continue
		}
		existedGroups.Insert(groupName)
		groupTcMap[groupName] = tc
	}

	// Calculate difference then update, delete or create
	toDelete := existedGroups.Difference(planGroups)
	err = am.deleteAutoscalingClusters(tc, toDelete.UnsortedList(), groupTcMap)
	if err != nil {
		return err
	}

	toUpdate := planGroups.Intersection(existedGroups)
	err = am.updateAutoscalingClusters(tac, toUpdate.UnsortedList(), groupTcMap, groupPlanMap)
	if err != nil {
		return err
	}

	toCreate := planGroups.Difference(existedGroups)
	err = am.createAutoscalingClusters(tc, tac, toCreate.UnsortedList(), groupPlanMap)
	if err != nil {
		return err
	}

	return nil
}

func (am *autoScalerManager) deleteAutoscalingClusters(tc *v1alpha1.TidbCluster, groupsToDelete []string, groupTcMap map[string]*v1alpha1.TidbCluster) error {
	var errs []error
	for _, group := range groupsToDelete {
		deleteTc := groupTcMap[group]

		// Remove cluster
		err := am.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Delete(deleteTc.Name, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}
	return errorutils.NewAggregate(errs)
}

func (am *autoScalerManager) updateAutoscalingClusters(tac *v1alpha1.TidbClusterAutoScaler, groupsToUpdate []string, groupTcMap map[string]*v1alpha1.TidbCluster, groupPlanMap map[string]pdapi.Plan) error {
	var errs []error
	for _, group := range groupsToUpdate {
		actual, oldTc, plan := groupTcMap[group].DeepCopy(), groupTcMap[group], groupPlanMap[group]

		switch plan.Component {
		case v1alpha1.TiKVMemberType.String():
			if tac.Spec.TiKV == nil || actual.Spec.TiKV.Replicas == int32(plan.Count) {
				continue
			}
			actual.Spec.TiKV.Replicas = int32(plan.Count)
		case v1alpha1.TiDBMemberType.String():
			if tac.Spec.TiDB == nil || actual.Spec.TiDB.Replicas == int32(plan.Count) {
				continue
			}
			actual.Spec.TiDB.Replicas = int32(plan.Count)
		default:
			errs = append(errs, fmt.Errorf("unexpected component %s for group %s in autoscaling plan", plan.Component, group))
			continue
		}

		_, err := am.tcControl.UpdateTidbCluster(actual, &actual.Status, &oldTc.Status)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}
	return errorutils.NewAggregate(errs)
}

func (am *autoScalerManager) createAutoscalingClusters(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, groupsToCreate []string, groupPlanMap map[string]pdapi.Plan) error {
	var errs []error
	for _, group := range groupsToCreate {
		plan := groupPlanMap[group]
		component := plan.Component
		var resources map[string]v1alpha1.AutoResource

		switch component {
		case v1alpha1.TiKVMemberType.String():
			if tac.Spec.TiKV == nil {
				continue
			}
			resources = getSpecResources(tac, v1alpha1.TiKVMemberType)
		case v1alpha1.TiDBMemberType.String():
			if tac.Spec.TiDB == nil {
				continue
			}
			resources = getSpecResources(tac, v1alpha1.TiDBMemberType)
		}

		resource, ok := resources[plan.ResourceType]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown resource type %v for group %s tac[%s/%s]", plan.ResourceType, plan.Labels[groupLabelKey], tac.Namespace, tac.Name))
			continue
		}
		requestsResourceList := corev1.ResourceList{
			corev1.ResourceCPU:    resource.CPU,
			corev1.ResourceMemory: resource.Memory,
		}

		limitsResourceList := corev1.ResourceList{
			corev1.ResourceCPU:    resource.CPU,
			corev1.ResourceMemory: resource.Memory,
		}

		autoTcName, err := genAutoClusterName(tac, component, plan.Labels, resource)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		autoTc := newAutoScalingCluster(tc, tac, autoTcName, component)
		autoTc.Labels[label.AutoScalingGroupLabelKey] = group

		switch component {
		case v1alpha1.TiKVMemberType.String():
			requestsResourceList[corev1.ResourceStorage] = resource.Storage

			autoTc.Spec.TiKV.Replicas = int32(plan.Count)
			autoTc.Spec.TiKV.ResourceRequirements = corev1.ResourceRequirements{
				Limits:   limitsResourceList,
				Requests: requestsResourceList,
			}

			// Assign Plan Labels
			for k, v := range plan.Labels {
				autoTc.Spec.TiKV.Config.Server.Labels[k] = v
			}
		case v1alpha1.TiDBMemberType.String():
			autoTc.Spec.TiDB.Replicas = int32(plan.Count)
			autoTc.Spec.TiDB.ResourceRequirements = corev1.ResourceRequirements{
				Limits:   limitsResourceList,
				Requests: requestsResourceList,
			}

			// Assign Plan Labels
			for k, v := range plan.Labels {
				autoTc.Spec.TiDB.Config.Labels[k] = v
			}
		}

		_, err = am.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(autoTc)
		if err != nil {
			errs = append(errs, err)
			continue
		}

	}
	return errorutils.NewAggregate(errs)
}

func (am *autoScalerManager) syncMonitor(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	if monitorRef := tc.Status.Monitor; monitorRef != nil {
		autoTcList, err := am.getAutoScaledClusters(tac, []v1alpha1.MemberType{v1alpha1.TiDBMemberType, v1alpha1.TiKVMemberType})
		if err != nil {
			return err
		}
		clusters := sets.String{}
		clusterMap := make(map[string]*v1alpha1.TidbCluster)
		for _, autoTc := range autoTcList {
			fullName := fmt.Sprintf("%s/%s", autoTc.Namespace, autoTc.Name)
			clusters.Insert(fullName)
			clusterMap[fullName] = autoTc
		}

		monitor, err := am.tmLister.TidbMonitors(monitorRef.Namespace).Get(monitorRef.Name)
		if err != nil {
			return err
		}

		monitoredClusters := sets.String{}
		newClusterRefs := make([]v1alpha1.TidbClusterRef, 0)
		for _, tcRef := range monitor.Spec.Clusters {
			ns := tcRef.Namespace
			if len(ns) == 0 {
				ns = monitor.Namespace
			}

			fullName := fmt.Sprintf("%s/%s", ns, tcRef.Name)
			monitoredClusters.Insert(fullName)
			tc, err := am.tcLister.TidbClusters(ns).Get(tcRef.Name)
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				} else {
					return err
				}
			}
			tacName, ok := tc.Labels[label.AutoInstanceLabelKey]
			if !ok || tacName != tac.Name {
				// Not controlled by this AutoScaler, keep it
				newClusterRefs = append(newClusterRefs, tcRef)
				continue
			}
		}

		updatedTm := monitor.DeepCopy()
		toAdd := clusters.Difference(monitoredClusters)
		for name := range toAdd {
			cluster := clusterMap[name]
			newClusterRefs = append(newClusterRefs,
				v1alpha1.TidbClusterRef{Namespace: cluster.Namespace, Name: cluster.Name})
		}
		updatedTm.Spec.Clusters = newClusterRefs

		err = am.updateTidbMonitor(updatedTm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (am *autoScalerManager) updateTidbMonitor(tm *v1alpha1.TidbMonitor) error {
	ns := tm.GetNamespace()
	tmName := tm.GetName()
	monitorSpec := tm.Spec.DeepCopy()

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
			tm.Spec = *monitorSpec
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
