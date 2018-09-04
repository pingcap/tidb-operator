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
	apps "k8s.io/api/apps/v1beta1"
)

type tikvUpgrader struct{}

// NewTiKVUpgrader returns a tikv Upgrader
func NewTiKVUpgrader() Upgrader {
	return &tikvUpgrader{}
}

func (tku *tikvUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
	} else {
		tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	}
	return nil
}

type fakeTiKVUpgrader struct{}

// NewFakeTiKVUpgrader returns a fake tikv upgrader
func NewFakeTiKVUpgrader() Upgrader {
	return &fakeTiKVUpgrader{}
}

func (tku *fakeTiKVUpgrader) Upgrade(tc *v1alpha1.TidbCluster, _ *apps.StatefulSet, _ *apps.StatefulSet) error {
	tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	return nil
}
