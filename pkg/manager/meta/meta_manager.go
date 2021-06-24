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

package meta

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
)

type metaManager struct {
	deps *controller.Dependencies
}

// NewMetaManager returns a *metaManager
func NewMetaManager(deps *controller.Dependencies) manager.Manager {
	return &metaManager{
		deps: deps,
	}
}

func (m *metaManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	instanceName := tc.GetInstanceName()

	l, err := label.New().Instance(instanceName).Selector()
	if err != nil {
		return err
	}
	pods, err := m.deps.PodLister.Pods(ns).List(l)
	if err != nil {
		return fmt.Errorf("metaManager.Sync: failed to list pods for cluster %s/%s, selector: %s, error: %v", ns, instanceName, l, err)
	}

	for _, pod := range pods {
		// update meta info for pod
		_, err := m.deps.PodControl.UpdateMetaInfo(tc, pod)
		if err != nil {
			return err
		}

		var mustUsePV bool
		switch pod.Labels[label.ComponentLabelKey] {
		case label.PDLabelVal,
			label.TiKVLabelVal,
			label.TiFlashLabelVal,
			label.PumpLabelVal:
			// Currently PD/TiKV/TiFlash/Pump must uses PV
			mustUsePV = true
		case label.TiDBLabelVal,
			label.TiCDCLabelVal:
			// Currently TiDB/TiCDC maybe uses PV
			mustUsePV = false
		default:
			// Skip syncing meta info for pod that doesn't use PV
			continue
		}

		// update meta info for pvc
		pvcs, err := util.ResolvePVCFromPod(pod, m.deps.PVCLister)
		if err != nil {
			if errors.IsNotFound(err) && !mustUsePV {
				continue
			}
			return err
		}
		for _, pvc := range pvcs {
			_, err = m.deps.PVCControl.UpdateMetaInfo(tc, pvc, pod)
			if err != nil {
				return err
			}
			if pvc.Spec.VolumeName == "" {
				continue
			}

			if m.deps.PVLister == nil {
				klog.V(4).Infof("Persistent volumes lister is unavailable, skip updating meta info for %s. This may be caused by no relevant permissions", pvc.Spec.VolumeName)
				continue
			}
			// update meta info for pv
			pv, err := m.deps.PVLister.Get(pvc.Spec.VolumeName)
			if err != nil {
				klog.Errorf("Get PV %s error: %v", pvc.Spec.VolumeName, err)
				return err
			}
			_, err = m.deps.PVControl.UpdateMetaInfo(tc, pv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var _ manager.Manager = &metaManager{}

type FakeMetaManager struct {
	err error
}

func NewFakeMetaManager() *FakeMetaManager {
	return &FakeMetaManager{}
}

func (m *FakeMetaManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeMetaManager) Sync(_ *v1alpha1.TidbCluster) error {
	return m.err
}
