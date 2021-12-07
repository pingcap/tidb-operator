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
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const (
	// Event Type
	EventTypeFailedSync  = "FailedSync"
	EventTypeSuccessSync = "SuccessSync"
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
	deps *controller.Dependencies,
	ngmMnger manager.TiDBNGMonitoringManager,
	reclaimPolicyManager ReclaimPolicyManager,
) ControlInterface {

	return &defaultTiDBNGMonitoringControl{
		deps:                 deps,
		ngmMnger:             ngmMnger,
		reclaimPolicyManager: reclaimPolicyManager,
	}
}

type defaultTiDBNGMonitoringControl struct {
	deps *controller.Dependencies

	ngmMnger             manager.TiDBNGMonitoringManager
	reclaimPolicyManager ReclaimPolicyManager
}

func (c *defaultTiDBNGMonitoringControl) Reconcile(tngm *v1alpha1.TidbNGMonitoring) error {
	// TODO: default and validate

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

		update, updateErr = c.deps.Clientset.PingcapV1alpha1().TidbNGMonitorings(ns).Update(context.TODO(), tngm, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("TidbNGMonitoring: [%s/%s] updated successfully", ns, name)
			return nil
		}

		klog.V(4).Infof("failed to update TidbNGMonitoring: [%s/%s], error: %v", ns, name, updateErr)

		if updated, err := c.deps.TiDBNGMonitoringLister.TidbNGMonitorings(ns).Get(name); err == nil {
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
