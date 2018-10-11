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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
)

const (
	// MaxResignDDLOwnerCount is the max regign DDL owner count
	MaxResignDDLOwnerCount = 3
)

type tidbUpgrader struct {
	tidbControl controller.TiDBControlInterface
}

// NewTiDBUpgrader returns a tidb Upgrader
func NewTiDBUpgrader(tidbControl controller.TiDBControlInterface) Upgrader {
	return &tidbUpgrader{tidbControl: tidbControl}
}

func (tdu *tidbUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}

	tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	setUpgradePartition(newSet, *oldSet.Spec.UpdateStrategy.RollingUpdate.Partition)

	if tc.Status.TiDB.StatefulSet.CurrentReplicas == 0 {
		return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb doesn't have old version pod to upgrade", ns, tcName)
	}

	if !tc.TiDBAllPodsStarted() {
		return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb pods are not all created", ns, tcName)
	}

	for i := tc.Status.TiDB.StatefulSet.Replicas; i > tc.Status.TiDB.StatefulSet.CurrentReplicas; i-- {
		if member, exist := tc.Status.TiDB.Members[tidbPodName(tcName, i-1)]; !exist || !member.Health {
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s tidb upgraded pods are not all ready", ns, tcName)
		}
	}

	upgradeOrdinal := tc.Status.TiDB.StatefulSet.CurrentReplicas - 1
	if member, exist := tc.Status.TiDB.Members[tidbPodName(tcName, upgradeOrdinal)]; exist && member.Health {
		err := tdu.tidbControl.ResignDDLOwner(tc, upgradeOrdinal)
		if err != nil && tc.Status.TiDB.ResignDDLOwnerFailCount < MaxResignDDLOwnerCount {
			tc.Status.TiDB.ResignDDLOwnerFailCount++
			return err
		}
	}

	tc.Status.TiDB.ResignDDLOwnerFailCount = 0
	setUpgradePartition(newSet, upgradeOrdinal)
	return nil
}

type fakeTiDBUpgrader struct{}

// NewFakeTiDBUpgrader returns a fake tidb upgrader
func NewFakeTiDBUpgrader() Upgrader {
	return &fakeTiDBUpgrader{}
}

func (ftdu *fakeTiDBUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	return nil
}
