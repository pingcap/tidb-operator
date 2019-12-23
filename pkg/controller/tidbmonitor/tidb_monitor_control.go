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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/errors"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

// ControlInterface reconciles TidbMonitor
type ControlInterface interface {
	// ReconcileTidbMonitor implements the reconcile logic of TidbMonitor
	ReconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error
}

// NewDefaultTidbMonitorControl returns a new instance of the default TidbMonitor ControlInterface
func NewDefaultTidbMonitorControl(recorder record.EventRecorder, ctrl controller.TypedControlInterface) ControlInterface {
	return &defaultTidbMonitorControl{recorder, ctrl}
}

type defaultTidbMonitorControl struct {
	recorder     record.EventRecorder
	typedControl controller.TypedControlInterface
}

func (tmc *defaultTidbMonitorControl) ReconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error {
	var errs []error
	oldStatus := tm.Status.DeepCopy()

	if err := tmc.reconcileTidbMonitor(tm); err != nil {
		errs = append(errs, err)
	}
	if apiequality.Semantic.DeepEqual(&tm.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	if _, err := tmc.tmControl.UpdateTidbCluster(tm.DeepCopy(), &tm.Status, oldStatus); err != nil {
		errs = append(errs, err)
	}
	// TODO: implementation
	return errors.NewAggregate(errs)
}

func (tmc *defaultTidbMonitorControl) reconcileTidbMonitor(tm *v1alpha1.TidbMonitor) error {

	// TODO: sync configmaps

	// TODO: sync secrets

	// TODO: sync Prometheus

	// TODO: sync Grafana

	return nil
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
