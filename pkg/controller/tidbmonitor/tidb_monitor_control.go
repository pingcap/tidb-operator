// Copyright 2019. PingCAP, Inc.
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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/monitor"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

// ControlInterface reconciles TidbMonitor
type ControlInterface interface {
	// ReconcileTidbMonitor implements the reconcile logic of TidbMonitor
	ReconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error
}

// NewDefaultTidbMonitorControl returns a new instance of the default TidbMonitor ControlInterface
func NewDefaultTidbMonitorControl(recorder record.EventRecorder, ctrl controller.TypedControlInterface, monitorManager monitor.MonitorManager) ControlInterface {
	return &defaultTidbMonitorControl{recorder, ctrl, monitorManager}
}

type defaultTidbMonitorControl struct {
	recorder       record.EventRecorder
	typedControl   controller.TypedControlInterface
	monitorManager monitor.MonitorManager
}

func (tmc *defaultTidbMonitorControl) ReconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error {
	var errs []error
	if err := tmc.reconcileTidbMonitor(tm); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}

func (tmc *defaultTidbMonitorControl) reconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error {

	return tmc.monitorManager.Sync(tm)
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

var _ ControlInterface = &FakeTidbMonitorControl{}
