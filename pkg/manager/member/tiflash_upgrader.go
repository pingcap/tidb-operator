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
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1"
)

type tiflashUpgrader struct {
	deps *controller.Dependencies
}

// NewTiFlashUpgrader returns a tiflash Upgrader
func NewTiFlashUpgrader(deps *controller.Dependencies) Upgrader {
	return &tiflashUpgrader{
		deps: deps,
	}
}

func (u *tiflashUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	//  Wait for PD, TiKV and TiDB to finish upgrade
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase ||
		tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
		return nil
	}
	return nil
}

type fakeTiFlashUpgrader struct{}

// NewFakeTiFlashUpgrader returns a fake tiflash upgrader
func NewFakeTiFlashUpgrader() Upgrader {
	return &fakeTiFlashUpgrader{}
}

func (_ *fakeTiFlashUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	return nil
}
