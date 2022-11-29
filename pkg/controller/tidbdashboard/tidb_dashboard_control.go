// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/defaulting"
	v1alpha1validation "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/validation"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// ReclaimPolicyManager abstracts the logic of reclaiming the pv object of TiDBDashboard.
type ReclaimPolicyManager interface {
	SyncTiDBDashboard(dashboard *v1alpha1.TidbDashboard) error
}

// ControlInterface abstracts the business logic for TiDBDashboard reconciliation.
type ControlInterface interface {
	Reconcile(*v1alpha1.TidbDashboard) error
}

func NewTiDBDashboardControl(
	deps *controller.Dependencies,
	dashboardManager manager.TiDBDashboardManager,
	tlsCertManager manager.TiDBDashboardManager,
	reclaimPolicyManager ReclaimPolicyManager,
	recorder record.EventRecorder,
) ControlInterface {

	return &defaultTiDBDashboardControl{
		deps:                 deps,
		recorder:             recorder,
		dashboardManager:     dashboardManager,
		tlsCertManager:       tlsCertManager,
		reclaimPolicyManager: reclaimPolicyManager,
	}
}

type defaultTiDBDashboardControl struct {
	deps     *controller.Dependencies
	recorder record.EventRecorder

	dashboardManager     manager.TiDBDashboardManager
	tlsCertManager       manager.TiDBDashboardManager
	reclaimPolicyManager ReclaimPolicyManager
}

func (c *defaultTiDBDashboardControl) Reconcile(td *v1alpha1.TidbDashboard) error {
	c.defaulting(td)
	if !c.validate(td) {
		return nil
	}

	oldStatus := td.Status.DeepCopy()

	if td.DeletionTimestamp != nil {
		return nil
	}

	var err error

	// Only the first TidbClusterRef is respected.
	tcRef := td.Spec.Clusters[0]
	tc, err := c.deps.TiDBClusterLister.TidbClusters(tcRef.Namespace).Get(tcRef.Name)
	if err != nil {
		return fmt.Errorf("get tc %s/%s failed: %s", tcRef.Namespace, tcRef.Name, err)
	}

	err = c.reclaimPolicyManager.SyncTiDBDashboard(td)
	if err != nil {
		return err
	}

	err = c.tlsCertManager.Sync(td, tc)
	if err != nil {
		return err
	}

	err = c.dashboardManager.Sync(td, tc)
	if err != nil {
		return err
	}

	if apiequality.Semantic.DeepEqual(&td.Status, oldStatus) {
		return nil
	}

	_, err = c.updateStatus(td.DeepCopy())
	if err != nil {
		return err
	}

	return nil
}

func (c *defaultTiDBDashboardControl) updateStatus(td *v1alpha1.TidbDashboard) (*v1alpha1.TidbDashboard, error) {
	var (
		ns     = td.GetNamespace()
		name   = td.GetName()
		status = td.Status.DeepCopy()
		update *v1alpha1.TidbDashboard
	)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		update, updateErr = c.deps.Clientset.PingcapV1alpha1().TidbDashboards(ns).UpdateStatus(context.TODO(), td, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("TidbDashboard: [%s/%s], update status successfully", ns, name)
			return nil
		}

		klog.V(4).Infof("TidbDashboard: [%s/%s], update status failed, error: %v", ns, name, updateErr)

		// If failed to update status, then:
		// get the latest TidbDashboard, override the status to local newest, prepare for next update.
		if updated, err := c.deps.TiDBDashboardLister.TidbDashboards(ns).Get(name); err == nil {
			td = updated.DeepCopy()
			td.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbDashboard %s/%s from lister: %v", ns, name, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("TidbDashboard: [%s/%s], failed to updateStatus, error: %v", ns, name, err)
	}

	return update, err
}

func (c *defaultTiDBDashboardControl) defaulting(td *v1alpha1.TidbDashboard) {
	defaulting.SetTidbDashboardDefault(td)
}

func (c *defaultTiDBDashboardControl) validate(td *v1alpha1.TidbDashboard) bool {
	errs := v1alpha1validation.ValidateTiDBDashboard(td)
	if len(errs) > 0 {
		aggregatedErr := errs.ToAggregate()
		klog.Errorf("tidb dashboard %s/%s is not valid and must be fixed first, aggregated error: %v", td.GetNamespace(), td.GetName(), aggregatedErr)
		c.recorder.Event(td, v1.EventTypeWarning, "FailedValidation", aggregatedErr.Error())
		return false
	}
	return true
}

type FakeTiDBDashboardControl struct {
	reconcile func(dashboard *v1alpha1.TidbDashboard) error
}

func (c *FakeTiDBDashboardControl) MockReconcile(reconcile func(*v1alpha1.TidbDashboard) error) {
	c.reconcile = reconcile
}

func (c *FakeTiDBDashboardControl) Reconcile(td *v1alpha1.TidbDashboard) error {
	if c.reconcile != nil {
		return c.reconcile(td)
	}
	return nil
}
