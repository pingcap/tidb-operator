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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
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
	tcList, err = am.deps.TiDBClusterLister.TidbClusters(tac.Spec.Cluster.Namespace).List(selector)
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
	err = am.deleteAutoscalingClusters(tac, tc, toDelete.UnsortedList(), groupTcMap)
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

func (am *autoScalerManager) deleteAutoscalingClusters(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster, groupsToDelete []string, groupTcMap map[string]*v1alpha1.TidbCluster) error {
	var errs []error
	for _, group := range groupsToDelete {
		deleteTc := groupTcMap[group]

		err := am.gracefullyDeleteTidbCluster(deleteTc)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if deleteTc.Spec.TiDB != nil {
			delete(tac.Status.TiDB, group)
		} else if deleteTc.Spec.TiKV != nil {
			delete(tac.Status.TiKV, group)
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
			if !checkAutoScaling(tac, v1alpha1.TiKVMemberType, group, actual.Spec.TiKV.Replicas, int32(plan.Count)) {
				continue
			}
			actual.Spec.TiKV.Replicas = int32(plan.Count)
		case v1alpha1.TiDBMemberType.String():
			if tac.Spec.TiDB == nil || actual.Spec.TiDB.Replicas == int32(plan.Count) {
				continue
			}
			if !checkAutoScaling(tac, v1alpha1.TiDBMemberType, group, actual.Spec.TiDB.Replicas, int32(plan.Count)) {
				continue
			}
			actual.Spec.TiDB.Replicas = int32(plan.Count)
		default:
			errs = append(errs, fmt.Errorf("unexpected component %s for group %s in autoscaling plan", plan.Component, group))
			continue
		}

		_, err := am.deps.TiDBClusterControl.UpdateTidbCluster(actual, &actual.Status, &oldTc.Status)
		if err != nil {
			klog.Errorf("tac[%s/%s] failed to update tc[%s/%s] for group %s, err: %v", tac.Namespace, tac.Name, actual.Namespace, actual.Name, group, err)
			errs = append(errs, err)
			continue
		}

		updateLastAutoScalingTimestamp(tac, plan.Component, group)
	}
	return errorutils.NewAggregate(errs)
}

func (am *autoScalerManager) createAutoscalingClusters(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, groupsToCreate []string, groupPlanMap map[string]pdapi.Plan) error {
	var errs []error
	for _, group := range groupsToCreate {
		plan := groupPlanMap[group]
		component := plan.Component
		resources := getSpecResources(tac, v1alpha1.MemberType(component))

		if resources == nil {
			errs = append(errs, fmt.Errorf("unknown component %s for group %s tac[%s/%s]", component, group, tac.Namespace, tac.Name))
			continue
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
				autoTc.Spec.TiKV.Config.Set("server.labels."+k, v)
			}
		case v1alpha1.TiDBMemberType.String():
			autoTc.Spec.TiDB.Replicas = int32(plan.Count)
			autoTc.Spec.TiDB.ResourceRequirements = corev1.ResourceRequirements{
				Limits:   limitsResourceList,
				Requests: requestsResourceList,
			}

			// Assign Plan Labels
			for k, v := range plan.Labels {
				autoTc.Spec.TiDB.Config.Set("labels."+k, v)
			}
		}

		_, err = am.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(autoTc)
		if err != nil {
			klog.Errorf("tac[%s/%s] failed to create autoscaling tc for group %s, err: %v", tac.Namespace, tac.Name, group, err)
			errs = append(errs, err)
			continue
		}

		updateLastAutoScalingTimestamp(tac, component, group)
	}
	return errorutils.NewAggregate(errs)
}
