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

package tidbmonitor

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/monitor"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// ControlInterface reconciles TidbMonitor
type ControlInterface interface {
	// ReconcileTidbMonitor implements the reconcile logic of TidbMonitor
	ReconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error

	// Update tidbmonitor status
	UpdateTidbMonitor(*v1alpha1.TidbMonitor) (*v1alpha1.TidbMonitor, error)
}

// NewDefaultTidbMonitorControl returns a new instance of the default TidbMonitor ControlInterface
func NewDefaultTidbMonitorControl(deps *controller.Dependencies, monitorManager monitor.MonitorManager) ControlInterface {
	return &defaultTidbMonitorControl{deps: deps, monitorManager: monitorManager}
}

type defaultTidbMonitorControl struct {
	deps           *controller.Dependencies
	monitorManager monitor.MonitorManager
}

func (c *defaultTidbMonitorControl) ReconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error {
	var errs []error
	if err := c.reconcileTidbMonitor(tm.DeepCopy()); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (c *defaultTidbMonitorControl) reconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error {
	var errs []error
	oldStatus := tm.Status.DeepCopy()
	if err := c.monitorManager.SyncMonitor(tm); err != nil {
		errs = append(errs, err)
	}

	if apiequality.Semantic.DeepEqual(&tm.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	klog.Error("test dm monitor --------333333")
	if _, err := c.UpdateTidbMonitor(tm.DeepCopy()); err != nil {
		errs = append(errs, err)
	}
	klog.Error("test dm monitor --------444444")

	return errorutils.NewAggregate(errs)
}

var _ ControlInterface = &defaultTidbMonitorControl{}

// FakeTidbMonitorControl is a fake TidbMonitor ControlInterface
type FakeTidbMonitorControl struct {
	err error
}

// NewFakeTidbMonitorControl returns a FakeBackupScheduleControl
func NewFakeTidbMonitorControl() *FakeTidbMonitorControl {
	return &FakeTidbMonitorControl{}
}

func (tmc *FakeTidbMonitorControl) SetReconcileTidbMonitorError(err error) {
	tmc.err = err
}

// CreateBackup adds the backup to BackupIndexer
func (tmc *FakeTidbMonitorControl) ReconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error {
	if tmc.err != nil {
		return tmc.err
	}
	return nil
}

// CreateBackup adds the backup to BackupIndexer
func (tmc *FakeTidbMonitorControl) UpdateTidbMonitor(tm *v1alpha1.TidbMonitor) (*v1alpha1.TidbMonitor, error) {
	return nil, nil
}

var _ ControlInterface = &FakeTidbMonitorControl{}

func (c *defaultTidbMonitorControl) UpdateTidbMonitor(tm *v1alpha1.TidbMonitor) (*v1alpha1.TidbMonitor, error) {
	ns := tm.GetNamespace()
	tmName := tm.GetName()

	status := tm.Status.DeepCopy()
	var update *v1alpha1.TidbMonitor

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		update, updateErr = c.deps.Clientset.PingcapV1alpha1().TidbMonitors(ns).Update(context.TODO(), tm, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("TidbMonitor: [%s/%s] updated successfully", ns, tmName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbMonitor: [%s/%s], error: %v", ns, tmName, updateErr)

		if updated, err := c.deps.TiDBMonitorLister.TidbMonitors(ns).Get(tmName); err == nil {
			// make a copy so we don't mutate the shared cache
			tm = updated.DeepCopy()
			tm.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbMonitor %s/%s from lister: %v", ns, tmName, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update TidbMonitor: [%s/%s], error: %v", ns, tmName, err)
	}
	return update, err
}
