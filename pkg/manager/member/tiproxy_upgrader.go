// Copyright 2018 PingCAP, Inc.
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

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	apps "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type tiproxyUpgrader struct {
	deps *controller.Dependencies
}

// NewTiProxyUpgrader returns a tiproxyUpgrader
func NewTiProxyUpgrader(deps *controller.Dependencies) Upgrader {
	return &tiproxyUpgrader{
		deps: deps,
	}
}

func (u *tiproxyUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if !tc.Status.TiProxy.Synced {
		return fmt.Errorf("tidbcluster: [%s/%s]'s tiproxy status sync failed, can not to be upgraded", ns, tcName)
	}
	if tc.Status.TiProxy.Phase == v1alpha1.ScalePhase {
		klog.Infof("TidbCluster: [%s/%s]'s tiproxy status is %v, can not upgrade tiproxy",
			ns, tcName, tc.Status.TiProxy.Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.TiProxy.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.TiProxy.StatefulSet.UpdateRevision == tc.Status.TiProxy.StatefulSet.CurrentRevision {
		return nil
	}

	mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for _i := len(podOrdinals) - 1; _i >= 0; _i-- {
		i := podOrdinals[_i]
		podName := PdPodName(tcName, i)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("gracefulUpgrade: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}

		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tiproxy pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.TiProxy.StatefulSet.UpdateRevision {
			if !podutil.IsPodReady(pod) {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded tiproxy pod: [%s] is not ready", ns, tcName, podName)
			}
			continue
		}

		mngerutils.SetUpgradePartition(newSet, i)
		return nil
	}
	return nil
}
