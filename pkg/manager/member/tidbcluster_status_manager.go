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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
)

type TidbClusterStatusManager struct {
	deps *controller.Dependencies
}

func NewTidbClusterStatusManager(deps *controller.Dependencies) *TidbClusterStatusManager {
	return &TidbClusterStatusManager{
		deps: deps,
	}
}

func (m *TidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	return m.syncTidbMonitorRefAndKey(tc)
}

func (m *TidbClusterStatusManager) syncTidbMonitorRefAndKey(tc *v1alpha1.TidbCluster) error {
	return m.syncAutoScalerRef(tc)
}

func (m *TidbClusterStatusManager) syncAutoScalerRef(tc *v1alpha1.TidbCluster) error {
	if tc.Status.AutoScaler == nil {
		klog.V(4).Infof("tc[%s/%s] autoscaler is empty", tc.Namespace, tc.Name)
		return nil
	}
	tacNamespace := tc.Status.AutoScaler.Namespace
	tacName := tc.Status.AutoScaler.Name
	tac, err := m.deps.TiDBClusterAutoScalerLister.TidbClusterAutoScalers(tacNamespace).Get(tacName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("tc[%s/%s] failed to find tac[%s/%s]", tc.Namespace, tc.Name, tacNamespace, tacName)
			tc.Status.AutoScaler = nil
			err = nil
		} else {
			err = fmt.Errorf("syncAutoScalerRef: failed to get tac %s/%s for cluster %s/%s, error: %s", tacNamespace, tacName, tc.GetNamespace(), tc.GetName(), err)
		}
		return err
	}
	if tac.Spec.Cluster.Name != tc.Name {
		klog.Infof("tc[%s/%s]'s target tac[%s/%s]'s cluster have been changed", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
		tc.Status.AutoScaler = nil
		return nil
	}
	if len(tac.Spec.Cluster.Namespace) < 1 {
		return nil
	}
	if tac.Spec.Cluster.Namespace != tc.Namespace {
		klog.Infof("tc[%s/%s]'s target tac[%s/%s]'s cluster namespace have been changed", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
		tc.Status.AutoScaler = nil
		return nil
	}
	return nil
}

type FakeTidbClusterStatusManager struct {
}

func NewFakeTidbClusterStatusManager() *FakeTidbClusterStatusManager {
	return &FakeTidbClusterStatusManager{}
}

func (f *FakeTidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	return nil
}
