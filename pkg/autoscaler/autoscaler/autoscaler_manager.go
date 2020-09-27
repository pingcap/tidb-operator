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
	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/api/errors"
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

func (m *autoScalerManager) Sync(tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.DeletionTimestamp != nil {
		return nil
	}

	tcName := tac.Spec.Cluster.Name
	// When Namespace in TidbClusterRef is omitted, we take tac's namespace as default
	if len(tac.Spec.Cluster.Namespace) < 1 {
		tac.Spec.Cluster.Namespace = tac.Namespace
	}

	tc, err := m.deps.TiDBClusterLister.TidbClusters(tac.Spec.Cluster.Namespace).Get(tcName)
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

	oldTc := tc.DeepCopy()
	if err := m.syncAutoScaling(tc, tac); err != nil {
		return err
	}

	return m.updateAutoScaling(oldTc, tac)
}

func (m *autoScalerManager) syncExternal(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, component v1alpha1.MemberType) error {
	switch component {
	case v1alpha1.TiDBMemberType:
		targetReplicas, err := query.ExternalService(tc, v1alpha1.TiDBMemberType, tac.Spec.TiDB.External.Endpoint, m.deps.KubeClientset)
		if err != nil {
			klog.Errorf("tac[%s/%s] 's query to the external endpoint got error: %v", tac.Namespace, tac.Name, err)
			return err
		}

		if tc.Spec.TiDB.Replicas == targetReplicas {
			return nil
		}

		updated := tc.DeepCopy()
		updated.Spec.TiDB.Replicas = targetReplicas
		if _, err = m.deps.TiDBClusterControl.UpdateTidbCluster(updated, &updated.Status, &tc.Status); err != nil {
			return err
		}
	case v1alpha1.TiKVMemberType:
		targetReplicas, err := query.ExternalService(tc, v1alpha1.TiKVMemberType, tac.Spec.TiKV.External.Endpoint, m.deps.KubeClientset)
		if err != nil {
			klog.Errorf("tac[%s/%s] 's query to the external endpoint got error: %v", tac.Namespace, tac.Name, err)
			return err
		}

		if tc.Spec.TiKV.Replicas == targetReplicas {
			return nil
		}

		updated := tc.DeepCopy()
		updated.Spec.TiKV.Replicas = targetReplicas
		if _, err = m.deps.TiDBClusterControl.UpdateTidbCluster(updated, &updated.Status, &tc.Status); err != nil {
			return err
		}
	}

	return nil
}

func (m *autoScalerManager) syncAutoScaling(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	// Sync from external endpoints if specified
	if tac.Spec.TiDB != nil && tac.Spec.TiDB.External != nil {
		if err := m.syncExternal(tc, tac, v1alpha1.TiDBMemberType); err != nil {
			return err
		}
	}

	if tac.Spec.TiKV != nil && tac.Spec.TiKV.External != nil {
		if err := m.syncExternal(tc, tac, v1alpha1.TiKVMemberType); err != nil {
			return err
		}
	}

	// Construct PD Auto-scaling strategy
	strategy := autoscalerToStrategy(tac)

	if len(strategy.Rules) != 0 {
		// Request PD for auto-scaling plans
		plans, err := controller.GetPDClient(m.deps.PDControl, tc).GetAutoscalingPlans(*strategy)
		if err != nil {
			klog.Errorf("cannot get auto-scaling plans for tac[%s/%s] err:%v", tac.Namespace, tac.Name, err)
			return err
		}

		// Apply auto-scaling plans
		if err := m.syncPlans(tc, tac, plans); err != nil {
			klog.Errorf("cannot apply autoscaling plans for tac[%s/%s] err:%v", tac.Namespace, tac.Name, err)
			return err
		}
	}

	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return nil
}

func (m *autoScalerManager) updateAutoScaling(oldTc *v1alpha1.TidbCluster,
	tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Annotations == nil {
		tac.Annotations = map[string]string{}
	}
	now := time.Now()
	tac.Annotations[label.AnnLastSyncingTimestamp] = fmt.Sprintf("%d", now.Unix())
	return m.updateTidbClusterAutoScaler(tac)
}

func (m *autoScalerManager) updateTidbClusterAutoScaler(tac *v1alpha1.TidbClusterAutoScaler) error {

	ns := tac.GetNamespace()
	tacName := tac.GetName()
	oldTac := tac.DeepCopy()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		_, updateErr = m.deps.Clientset.PingcapV1alpha1().TidbClusterAutoScalers(ns).Update(tac)
		if updateErr == nil {
			klog.Infof("TidbClusterAutoScaler: [%s/%s] updated successfully", ns, tacName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbClusterAutoScaler: [%s/%s], error: %v", ns, tacName, updateErr)
		if updated, err := m.deps.TiDBClusterAutoScalerLister.TidbClusterAutoScalers(ns).Get(tacName); err == nil {
			// make a copy so we don't mutate the shared cache
			tac = updated.DeepCopy()
			tac.Annotations = oldTac.Annotations
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
