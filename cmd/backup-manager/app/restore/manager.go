// Copyright 2019 PingCAP, Inc.
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

package restore

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

type Manager struct {
	restoreLister listers.RestoreLister
	StatusUpdater controller.RestoreConditionUpdaterInterface
	Options
}

// NewManager return a RestoreManager
func NewManager(
	restoreLister listers.RestoreLister,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreOpts Options) *Manager {
	return &Manager{
		restoreLister,
		statusUpdater,
		restoreOpts,
	}
}

// ProcessRestore used to process the restore logic
func (rm *Manager) ProcessRestore() error {
	restore, err := rm.restoreLister.Restores(rm.Namespace).Get(rm.RestoreName)
	if err != nil {
		klog.Errorf("can't find cluster %s restore %s CRD object, err: %v", rm, rm.RestoreName, err)
		return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetRestoreCRFailed",
			Message: err.Error(),
		})
	}
	if restore.Spec.BR == nil {
		return fmt.Errorf("no br config in %s", rm)
	}

	return rm.performRestore(restore.DeepCopy())
}

func (rm *Manager) performRestore(restore *v1alpha1.Restore) error {
	started := time.Now()

	err := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreRunning,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	if err := rm.restoreData(restore); err != nil {
		klog.Errorf("restore cluster %s from %s failed, err: %s", rm, restore.Spec.Type, err)
		return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "RestoreDataFromRemoteFailed",
			Message: err.Error(),
		})
	}
	klog.Infof("restore cluster %s from %s succeed", rm, restore.Spec.Type)

	finish := time.Now()
	restore.Status.TimeStarted = metav1.Time{Time: started}
	restore.Status.TimeCompleted = metav1.Time{Time: finish}

	return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
}
