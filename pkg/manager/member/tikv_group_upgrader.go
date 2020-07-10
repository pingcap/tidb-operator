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

package member

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog"
)

type tikvGroupUpgrader struct {
}

// NewTiKVGroupUpgrader returns a tikvgroup Upgrader
func NewTiKVGroupUpgrader() TiKVGroupUpgrader {
	return &tikvGroupUpgrader{}
}

func (tku *tikvGroupUpgrader) Upgrade(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tg.GetNamespace()
	tcName := tc.GetName()
	tgName := tg.GetName()

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tg.Scaling() {
		klog.Infof("TidbCluster: [%s/%s]'s pd status is %v, tg[%s] status is %v, can not upgrade tikv",
			ns, tcName, tc.Status.PD.Phase, tg.Name, tg.Status.Phase)
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	if !tg.Status.Synced {
		return fmt.Errorf("TiKVGroup: [%s/%s]'s tikv status sync failed, can not to be upgraded", ns, tgName)
	}

	tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	if !templateEqual(newSet, oldSet) {
		return nil
	}

	if tc.Status.TiKV.StatefulSet.UpdateRevision == tc.Status.TiKV.StatefulSet.CurrentRevision {
		return nil
	}

	if oldSet.Spec.UpdateStrategy.Type == apps.OnDeleteStatefulSetStrategyType || oldSet.Spec.UpdateStrategy.RollingUpdate == nil {
		// Manually bypass tidb-operator to modify statefulset directly, such as modify tikv statefulset's RollingUpdate strategy to OnDelete strategy,
		// or set RollingUpdate to nil, skip tidb-operator's rolling update logic in order to speed up the upgrade in the test environment occasionally.
		// If we encounter this situation, we will let the native statefulset controller do the upgrade completely, which may be unsafe for upgrading tikv.
		// Therefore, in the production environment, we should try to avoid modifying the tikv statefulset update strategy directly.
		newSet.Spec.UpdateStrategy = oldSet.Spec.UpdateStrategy
		klog.Warningf("tidbcluster: [%s/%s] tikv statefulset %s UpdateStrategy has been modified manually", ns, tcName, oldSet.GetName())
		return nil
	}

	if controller.PodWebhookEnabled {
		setUpgradePartition(newSet, 0)
		return nil
	}

	return fmt.Errorf("tikvgroup[%s/%s] failed to upgrade, need to enable pod webhook", ns, tg.Name)
}
