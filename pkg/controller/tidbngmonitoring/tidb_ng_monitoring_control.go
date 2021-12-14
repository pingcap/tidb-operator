// Copyright 2021 PingCAP, Inc.
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

package tidbngmonitoring

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1alpha1validation "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/validation"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/manager"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

type ReclaimPolicyManager interface {
	SyncTiDBNGMonitoring(monitor *v1alpha1.TidbNGMonitoring) error
}

// ControlInterface provide function about control TidbNGMonitoring
type ControlInterface interface {
	// Reconcile a TidbNGMonitoring
	Reconcile(*v1alpha1.TidbNGMonitoring) error

	// Update a TidbNGMonitoring
	Update(*v1alpha1.TidbNGMonitoring) (*v1alpha1.TidbNGMonitoring, error)
}

func NewDefaultTiDBNGMonitoringControl(
	cli versioned.Interface,
	lister listers.TidbNGMonitoringLister,
	ngmMnger manager.TiDBNGMonitoringManager,
	reclaimPolicyManager ReclaimPolicyManager,
	recorder record.EventRecorder,
) *defaultTiDBNGMonitoringControl {

	return &defaultTiDBNGMonitoringControl{
		cli:                  cli,
		lister:               lister,
		recorder:             recorder,
		ngmMnger:             ngmMnger,
		reclaimPolicyManager: reclaimPolicyManager,
	}
}

type defaultTiDBNGMonitoringControl struct {
	cli      versioned.Interface
	lister   listers.TidbNGMonitoringLister
	recorder record.EventRecorder

	ngmMnger             manager.TiDBNGMonitoringManager
	reclaimPolicyManager ReclaimPolicyManager
}

func (c *defaultTiDBNGMonitoringControl) Reconcile(tngm *v1alpha1.TidbNGMonitoring) error {
	if !c.validate(tngm) {
		return nil // fatal error, no need to retry on invalid object
	}

	var errs []error

	oldStatus := tngm.Status.DeepCopy()

	// reconcile
	err := c.reconcile(tngm)
	if err != nil {
		errs = append(errs, err)
	}

	if apiequality.Semantic.DeepEqual(&tngm.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}

	// update resource
	_, err = c.Update(tngm.DeepCopy())
	if err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (c *defaultTiDBNGMonitoringControl) reconcile(tngm *v1alpha1.TidbNGMonitoring) error {
	if tngm.DeletionTimestamp != nil {
		return nil
	}

	// reoncile reclaim policy of pvc
	err := c.reclaimPolicyManager.SyncTiDBNGMonitoring(tngm)
	if err != nil {
		return err
	}

	// reconcile ng monitoring
	err = c.ngmMnger.Sync(tngm)
	if err != nil {
		return err
	}

	return nil
}

func (c *defaultTiDBNGMonitoringControl) Update(tngm *v1alpha1.TidbNGMonitoring) (*v1alpha1.TidbNGMonitoring, error) {
	var (
		ns     string                           = tngm.GetNamespace()
		name   string                           = tngm.GetName()
		status *v1alpha1.TidbNGMonitoringStatus = tngm.Status.DeepCopy()
		update *v1alpha1.TidbNGMonitoring
	)

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error

		update, updateErr = c.cli.PingcapV1alpha1().TidbNGMonitorings(ns).UpdateStatus(context.TODO(), tngm, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("TidbNGMonitoring: [%s/%s] updated successfully", ns, name)
			return nil
		}

		klog.V(4).Infof("failed to update TidbNGMonitoring: [%s/%s], error: %v", ns, name, updateErr)

		if updated, err := c.lister.TidbNGMonitorings(ns).Get(name); err == nil {
			// make a copy so we don't mutate the shared cache
			tngm = updated.DeepCopy()
			tngm.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbNGMonitoring %s/%s from lister: %v", ns, name, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update TidbMonTiDBNGMonitoringitor: [%s/%s], error: %v", ns, name, err)
	}
	return update, err
}

func (c *defaultTiDBNGMonitoringControl) validate(tngm *v1alpha1.TidbNGMonitoring) bool {
	errs := v1alpha1validation.ValidateTiDBNGMonitoring(tngm)
	if len(errs) > 0 {
		aggregatedErr := errs.ToAggregate()
		klog.Errorf("tidb ng monitoring %s/%s is not valid and must be fixed first, aggregated error: %v", tngm.GetNamespace(), tngm.GetName(), aggregatedErr)
		c.recorder.Event(tngm, v1.EventTypeWarning, "FailedValidation", aggregatedErr.Error())
		return false
	}
	return true
}

type FakeTiDBNGMonitoringControl struct {
	reconcile func(*v1alpha1.TidbNGMonitoring) error

	update func(*v1alpha1.TidbNGMonitoring) (*v1alpha1.TidbNGMonitoring, error)
}

func (c *FakeTiDBNGMonitoringControl) MockReconcile(reconcile func(*v1alpha1.TidbNGMonitoring) error) {
	c.reconcile = reconcile
}

func (c *FakeTiDBNGMonitoringControl) MockUpdate(update func(*v1alpha1.TidbNGMonitoring) (*v1alpha1.TidbNGMonitoring, error)) {
	c.update = update
}

func (c *FakeTiDBNGMonitoringControl) Reconcile(tngm *v1alpha1.TidbNGMonitoring) error {
	if c.reconcile != nil {
		return c.reconcile(tngm)
	}
	return nil
}

func (c *FakeTiDBNGMonitoringControl) Update(tngm *v1alpha1.TidbNGMonitoring) (*v1alpha1.TidbNGMonitoring, error) {
	if c.update != nil {
		return c.update(tngm)
	}
	return tngm, nil
}
