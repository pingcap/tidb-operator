// Copyright 2023 PingCAP, Inc.
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

package member

import (
	"fmt"
	"strings"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/third_party/k8s"
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
)

type pdMSUpgrader struct {
	deps *controller.Dependencies
}

// NewPDMSUpgrader returns a PD Micro Service Upgrader
func NewPDMSUpgrader(deps *controller.Dependencies) Upgrader {
	return &pdMSUpgrader{
		deps: deps,
	}
}

func (u *pdMSUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	return u.gracefulUpgrade(tc, oldSet, newSet)
}

func (u *pdMSUpgrader) gracefulUpgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if tc.Status.PDMS == nil {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pdMS status is nil, can not to be upgraded", ns, tcName)
	}

	componentName := controller.PDMSTrimName(newSet.Name)
	klog.Infof("gracefulUpgrade pdMS trim name, componentName: %s", componentName)
	if tc.Status.PDMS[componentName] == nil {
		tc.Status.PDMS[componentName] = &v1alpha1.PDMSStatus{Name: componentName}
		return fmt.Errorf("tidbcluster: [%s/%s]'s pdMS component is nil, can not to be upgraded, component: %s", ns, tcName, componentName)
	}
	if !tc.Status.PDMS[componentName].Synced {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pdMS status sync failed, can not to be upgraded, component: %s", ns, tcName, componentName)
	}
	oldTrimName := controller.PDMSTrimName(oldSet.Name)
	if oldTrimName != componentName {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pdMS oldTrimName is %s, not equal to componentName: %s", ns, tcName, oldTrimName, componentName)
	}
	klog.Infof("gracefulUpgrade pdMS trim name, oldTrimName: %s", oldTrimName)
	if tc.PDMSScaling(oldTrimName) {
		klog.Infof("TidbCluster: [%s/%s]'s pdMS status is %v, can not upgrade pdMS",
			ns, tcName, tc.Status.PDMS[componentName].Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.PDMS[componentName].Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify pd statefulset's RollingUpdate straregy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading pdMS.
		// Therefore, in the production environment, we should try to avoid modifying the pd statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] pdMS statefulset %s UpdateStrategy has been modified manually, componentName: %s", ns, tcName, oldSet.GetName(), componentName)
		return nil
	}

	mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := PDMSPodName(tcName, i, oldTrimName)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}

		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pdMS pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.PDMS[componentName].StatefulSet.UpdateRevision {
			if !k8s.IsPodReady(pod) {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded pdMS pod: [%s] is not ready", ns, tcName, podName)
			}

			var exist bool
			for _, member := range tc.Status.PDMS[componentName].Members {
				if strings.Contains(member, podName) {
					exist = true
				}
			}
			if !exist {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pdMS upgraded pod: [%s] is not exist, all members: %v",
					ns, tcName, podName, tc.Status.PDMS[componentName].Members)
			}
			continue
		}
		mngerutils.SetUpgradePartition(newSet, i)
		return nil
	}

	return nil
}

type fakePDMSUpgrader struct{}

// NewFakePDMSUpgrader returns a fakePDUpgrader
func NewFakePDMSUpgrader() Upgrader {
	return &fakePDMSUpgrader{}
}

func (u *fakePDMSUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if tc.Status.PDMS == nil {
		return fmt.Errorf("tidbcluster: [%s/%s]'s pdMS status is nil, can not to be upgraded", tc.GetNamespace(), tc.GetName())
	}

	componentName := controller.PDMSTrimName(newSet.Spec.ServiceName)
	if tc.Status.PDMS[componentName] == nil {
		tc.Status.PDMS[componentName] = &v1alpha1.PDMSStatus{Name: componentName}
		return fmt.Errorf("tidbcluster: [%s/%s]'s pdMS component is nil, can not to be upgraded, component: %s", tc.GetNamespace(), tc.GetName(), componentName)
	}

	if !tc.Status.PDMS[componentName].Synced {
		return fmt.Errorf("tidbcluster: pd ms status sync failed, can not to be upgraded")
	}
	println("fake pd ms upgrade")
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}
