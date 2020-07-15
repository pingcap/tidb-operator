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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/klog"
)

type TikvGroupUpgrader struct {
	tikvupgrader TiKVUpgrader
}

// NewTiKVGroupUpgrader returns a tikvgroup Upgrader
func NewTiKVGroupUpgrader(tikvUpgrader TiKVUpgrader) *TikvGroupUpgrader {
	return &TikvGroupUpgrader{
		tikvupgrader: tikvUpgrader,
	}
}

func (tku *TikvGroupUpgrader) Upgrade(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tg.GetNamespace()
	tcName := tc.GetName()

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
	return tku.tikvupgrader.Upgrade(tg, oldSet, newSet)
}
