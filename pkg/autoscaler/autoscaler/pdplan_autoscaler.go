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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

const groupLabelKey = "group"

func (am *autoScalerManager) syncPlans(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, plans []pdapi.Plan) error {
	planGroups := sets.String{}
	groupPlanMap := make(map[string]pdapi.Plan)
	for _, plan := range plans {
		groupName := plan.Labels[groupLabelKey]
		planGroups.Insert(groupName)
		groupPlanMap[groupName] = plan
	}

	// Filter all autoscaled TidbClusters
	requirement, err := labels.NewRequirement(label.AutoScalingGroupLabelKey, selection.Exists, nil)
	if err != nil {
		return err
	}
	selector := labels.NewSelector().Add(*requirement)
	tcList, err := am.tcLister.TidbClusters(tc.Namespace).List(selector)
	if err != nil {
		return err
	}

	existedGroups := sets.String{}
	groupTcMap := make(map[string]*v1alpha1.TidbCluster)
	for _, tc := range tcList {
		groupName := tc.Labels[label.AutoScalingGroupLabelKey]
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
	// TODO in next PR
	return nil
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
	// TODO in next PR
	return nil
}
