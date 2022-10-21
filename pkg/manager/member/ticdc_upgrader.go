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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type ticdcUpgrader struct {
	deps *controller.Dependencies
}

// NewTiCDCUpgrader returns a ticdc Upgrader
func NewTiCDCUpgrader(deps *controller.Dependencies) Upgrader {
	return &ticdcUpgrader{
		deps: deps,
	}
}

func (u *ticdcUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {

	// return nil when scale replicas to 0
	if tc.Spec.TiCDC.Replicas == int32(0) {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.PD.Phase == v1alpha1.ScalePhase ||
		tc.Status.TiKV.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.ScalePhase ||
		tc.Status.TiFlash.Phase == v1alpha1.UpgradePhase || tc.Status.TiFlash.Phase == v1alpha1.ScalePhase ||
		tc.Status.Pump.Phase == v1alpha1.UpgradePhase || tc.Status.Pump.Phase == v1alpha1.ScalePhase ||
		tc.Status.TiDB.Phase == v1alpha1.UpgradePhase || tc.Status.TiDB.Phase == v1alpha1.ScalePhase {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %s, "+
			"tikv status is %s, tiflash status is %s, pump status is %s, "+
			"tidb status is %s, can not upgrade ticdc",
			ns, tcName,
			tc.Status.PD.Phase, tc.Status.TiKV.Phase, tc.Status.TiFlash.Phase,
			tc.Status.Pump.Phase, tc.Status.TiDB.Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.TiCDC.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.TiCDC.StatefulSet.UpdateRevision == tc.Status.TiCDC.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify ticdc statefulset's RollingUpdate strategy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tidb.
		// Therefore, in the production environment, we should try to avoid modifying the tidb statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] ticdc statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	mngerutils.SetUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)
	podOrdinals := helper.GetPodOrdinals(*oldSet.Spec.Replicas, oldSet).List()
	for i := len(podOrdinals) - 1; i >= 0; i-- {
		ordinal := podOrdinals[i]
		podName := ticdcPodName(tcName, ordinal)
		pod, err := u.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return fmt.Errorf("ticdcUpgrader.Upgrade: failed to get pod %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
		}
		revision, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s ticdc pod: [%s] has no label: %s", ns, tcName, podName, apps.ControllerRevisionHashLabelKey)
		}

		if revision == tc.Status.TiCDC.StatefulSet.UpdateRevision {
			if !podutil.IsPodReady(pod) {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s upgraded ticdc pod: [%s] is not ready", ns, tcName, podName)
			}
			if _, exist := tc.Status.TiCDC.Captures[podName]; !exist {
				return controller.RequeueErrorf("tidbcluster: [%s/%s]'s ticdc upgraded pod: [%s] is not ready", ns, tcName, podName)
			}
			continue
		}

		support, err := isTiCDCPodSupportGracefulUpgrade(tc, u.deps.CDCControl, u.deps.PodControl, pod, ordinal, "Upgrade")
		if err != nil {
			return err
		}
		if support {
			err = gracefulDrainTiCDC(tc, u.deps.CDCControl, u.deps.PodControl, pod, ordinal, "Upgrade")
			if err != nil {
				return err
			}
			klog.Infof("ticdcUpgrade.Upgrade: %s graceful drain TiCDC complete in cluster %s/%s", podName, tc.GetNamespace(), tc.GetName())
			// To prevent TiCDC service disruption, we need to resign owner
			// gracefully from the next pod that is going to be upgraded.
			// If the current pod is the last one to upgrade, skip resign owner.
			hasNext := i-1 >= 0
			if hasNext {
				nextOrd := podOrdinals[i-1]
				nextPodName := ticdcPodName(tcName, nextOrd)
				klog.Infof("ticdcUpgrade.Upgrade: try to graceful resign owner from the next ticdc pod %s in cluster %s/%s", nextPodName, tc.GetNamespace(), tc.GetName())
				err = gracefulResignOwnerTiCDC(tc, u.deps.CDCControl, u.deps.PodControl, pod, nextPodName, nextOrd, "Upgrade")
				if err != nil {
					return err
				}
				klog.Infof("ticdcUpgrade.Upgrade: %s graceful resign owner complete in cluster %s/%s", nextPodName, tc.GetNamespace(), tc.GetName())
			}
			klog.Infof("ticdcUpgrade.Upgrade: %s graceful shutdown complete in cluster %s/%s", podName, tc.GetNamespace(), tc.GetName())
		}

		mngerutils.SetUpgradePartition(newSet, ordinal)
		return nil
	}

	return nil
}
