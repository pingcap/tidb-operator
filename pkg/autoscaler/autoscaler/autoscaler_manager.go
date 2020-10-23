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

package autoscaler

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/query"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

type autoScalerManager struct {
	deps *controller.Dependencies
}

func NewAutoScalerManager(deps *controller.Dependencies) *autoScalerManager {
	return &autoScalerManager{
		deps: deps,
	}
}

func (am *autoScalerManager) Sync(tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.DeletionTimestamp != nil {
		return nil
	}

	tcName := tac.Spec.Cluster.Name
	// When Namespace in TidbClusterRef is omitted, we take tac's namespace as default
	if len(tac.Spec.Cluster.Namespace) < 1 {
		tac.Spec.Cluster.Namespace = tac.Namespace
	}

	tc, err := am.deps.TiDBClusterLister.TidbClusters(tac.Spec.Cluster.Namespace).Get(tcName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Target TidbCluster Ref is deleted, empty the auto-scaling status
			return nil
		}
		return err
	}

	defaultTAC(tac, tc)
	if err := validateTAC(tac); err != nil {
		klog.Errorf("invalid spec tac[%s/%s]: %s", tac.Namespace, tac.Name, err.Error())
		return nil
	}

	updatedTac := tac.DeepCopy()

	if err := am.syncAutoScaling(tc, updatedTac); err != nil {
		return err
	}

	return am.updateTidbClusterAutoScaler(updatedTac)
}

func (am *autoScalerManager) syncExternal(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) error {
	var cfg *v1alpha1.ExternalConfig
	switch component {
	case v1alpha1.TiDBMemberType:
		cfg = tac.Spec.TiDB.External
	case v1alpha1.TiKVMemberType:
		cfg = tac.Spec.TiKV.External
	}

	targetReplicas, err := query.ExternalService(tc, component, cfg.Endpoint, am.deps.KubeClientset)
	if err != nil {
		klog.Errorf("tac[%s/%s]'s query to the external endpoint for component %s got error: %v", tac.Namespace, tac.Name, component.String(), err)
		return err
	}

	if targetReplicas > cfg.MaxReplicas {
		targetReplicas = cfg.MaxReplicas
	}

	return am.syncExternalResult(tc, tac, component, targetReplicas)
}

func (am *autoScalerManager) syncPD(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) error {
	strategy := autoscalerToStrategy(tac, component)
	// Request PD for auto-scaling plans
	plans, err := controller.GetPDClient(am.deps.PDControl, tc).GetAutoscalingPlans(*strategy)
	if err != nil {
		klog.Errorf("tac[%s/%s] cannot get auto-scaling plans for component %v err:%v", tac.Namespace, tac.Name, component, err)
		return err
	}

	// Apply auto-scaling plans
	if err := am.syncPlans(tc, tac, plans, component); err != nil {
		klog.Errorf("tac[%s/%s] cannot apply autoscaling plans for component %v err:%v", tac.Namespace, tac.Name, component, err)
		return err
	}
	return nil
}

func (am *autoScalerManager) syncAutoScaling(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	var errs []error
	if tac.Spec.TiDB != nil {
		if tac.Spec.TiDB.External != nil {
			if err := am.syncExternal(tc, tac, v1alpha1.TiDBMemberType); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err := am.syncPD(tc, tac, v1alpha1.TiDBMemberType); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if tac.Spec.TiKV != nil {
		if tac.Spec.TiKV.External != nil {
			if err := am.syncExternal(tc, tac, v1alpha1.TiKVMemberType); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err := am.syncPD(tc, tac, v1alpha1.TiKVMemberType); err != nil {
				errs = append(errs, err)
			}
		}
	}

	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return errorutils.NewAggregate(errs)
}

func (am *autoScalerManager) gracefullyDeleteTidbCluster(deleteTc *v1alpha1.TidbCluster) error {
	// Remove cluster
	// If there are TiKV pods, delete the cluster gracefully because we need to transfer data
	if deleteTc.Spec.TiKV != nil {
		// The TC is not shutting down, set replicas to 0 to trigger data transfer
		if deleteTc.Spec.TiKV.Replicas != 0 {
			cloned := deleteTc.DeepCopy()
			cloned.Spec.TiKV.Replicas = 0
			_, err := am.deps.TiDBClusterControl.UpdateTidbCluster(cloned, &cloned.Status, &deleteTc.Status)
			return err
		}

		// The TC is shutting down, check for its status if all pods have been deleted
		if deleteTc.Status.TiKV.StatefulSet != nil && deleteTc.Status.TiKV.StatefulSet.Replicas != 0 {
			// Still shutting down, do nothing
			return nil
		}

		// The TC has scaled in, fall through the code to delete it
	}

	return am.deps.Clientset.PingcapV1alpha1().TidbClusters(deleteTc.Namespace).Delete(deleteTc.Name, nil)
}

func (am *autoScalerManager) updateTidbClusterAutoScaler(tac *v1alpha1.TidbClusterAutoScaler) error {
	ns := tac.GetNamespace()
	tacName := tac.GetName()
	oldTac := tac.DeepCopy()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		_, updateErr = am.deps.Clientset.PingcapV1alpha1().TidbClusterAutoScalers(ns).Update(tac)
		if updateErr == nil {
			klog.Infof("TidbClusterAutoScaler: [%s/%s] updated successfully", ns, tacName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbClusterAutoScaler: [%s/%s], error: %v", ns, tacName, updateErr)
		if updated, err := am.deps.TiDBClusterAutoScalerLister.TidbClusterAutoScalers(ns).Get(tacName); err == nil {
			// make a copy so we don't mutate the shared cache
			tac = updated.DeepCopy()
			tac.Status = oldTac.Status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbClusterAutoScaler %s/%s from lister: %v", ns, tacName, err))
		}
		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update TidbClusterAutoScaler: [%s/%s], error: %v", ns, tacName, err)
	}
	return err
}

func updateLastAutoScalingTimestamp(tac *v1alpha1.TidbClusterAutoScaler, memberType string, group string) {
	switch memberType {
	case v1alpha1.TiKVMemberType.String():
		if tac.Status.TiKV == nil {
			tac.Status.TiKV = map[string]v1alpha1.TikvAutoScalerStatus{}
		}
		status := tac.Status.TiKV[group]
		status.LastAutoScalingTimestamp = &metav1.Time{Time: time.Now()}
		tac.Status.TiKV[group] = status
	case v1alpha1.TiDBMemberType.String():
		if tac.Status.TiDB == nil {
			tac.Status.TiDB = map[string]v1alpha1.TidbAutoScalerStatus{}
		}
		status := tac.Status.TiDB[group]
		status.LastAutoScalingTimestamp = &metav1.Time{Time: time.Now()}
		tac.Status.TiDB[group] = status
	}
}
