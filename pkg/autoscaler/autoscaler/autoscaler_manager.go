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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

type autoScalerManager struct {
	kubecli   kubernetes.Interface
	cli       versioned.Interface
	tcLister  v1alpha1listers.TidbClusterLister
	tmLister  v1alpha1listers.TidbMonitorLister
	tcControl controller.TidbClusterControlInterface
	pdControl pdapi.PDControlInterface
	taLister  v1alpha1listers.TidbClusterAutoScalerLister
	stsLister appslisters.StatefulSetLister
	recorder  record.EventRecorder
}

func NewAutoScalerManager(
	kubecli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) *autoScalerManager {
	tcLister := informerFactory.Pingcap().V1alpha1().TidbClusters().Lister()
	return &autoScalerManager{
		kubecli:   kubecli,
		cli:       cli,
		tcLister:  tcLister,
		tcControl: controller.NewRealTidbClusterControl(cli, tcLister, recorder),
		pdControl: pdapi.NewDefaultPDControl(kubecli),
		taLister:  informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers().Lister(),
		tmLister:  informerFactory.Pingcap().V1alpha1().TidbMonitors().Lister(),
		stsLister: kubeInformerFactory.Apps().V1().StatefulSets().Lister(),
		recorder:  recorder,
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

	tc, err := am.tcLister.TidbClusters(tac.Spec.Cluster.Namespace).Get(tcName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Target TidbCluster Ref is deleted, empty the auto-scaling status
			return nil
		}
		return err
	}
	permitted, err := am.checkAutoScalerRef(tc, tac)
	if err != nil {
		klog.Error(err)
		return err
	}
	if !permitted {
		klog.Infof("tac[%s/%s]'s auto-scaling is no permitted", tac.Namespace, tac.Name)
		return nil
	}

	oldTc := tc.DeepCopy()
	if err := am.syncAutoScaling(tc, tac); err != nil {
		return err
	}
	if err := am.syncTidbClusterReplicas(tac, tc, oldTc); err != nil {
		return err
	}
	return am.updateAutoScaling(oldTc, tac)
}

func (am *autoScalerManager) syncAutoScaling(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	defaultTAC(tac)

	// Construct PD Auto-scaling strategy
	strategy := autoscalerToStrategy(tac)

	// Request PD for auto-scaling plans
	plans, err := controller.GetPDClient(am.pdControl, tc).GetAutoscalingPlans(*strategy)
	if err != nil {
		klog.Errorf("cannot get auto-scaling plans %v", err)
		return err
	}

	// Distinguish plans
	plansMap := make(map[string][]pdapi.Plan)
	for _, plan := range plans {
		plansMap[plan.Component] = append(plansMap[plan.Component], plan)
	}

	oldTikvReplicas := tc.Spec.TiKV.Replicas
	if err := am.syncTiKV(tc, tac); err != nil {
		tc.Spec.TiKV.Replicas = oldTikvReplicas
		klog.Errorf("tac[%s/%s] tikv sync failed, continue to sync next, err:%v", tac.Namespace, tac.Name, err)
	}
	oldTidbReplicas := tc.Spec.TiDB.Replicas
	if err := am.syncTiDB(tc, tac); err != nil {
		tc.Spec.TiDB.Replicas = oldTidbReplicas
		klog.Errorf("tac[%s/%s] tidb sync failed, continue to sync next, err:%v", tac.Namespace, tac.Name, err)
	}
	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return nil
}

func (am *autoScalerManager) syncTidbClusterReplicas(tac *v1alpha1.TidbClusterAutoScaler, tc *v1alpha1.TidbCluster, oldTc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiDB.Replicas == oldTc.Spec.TiDB.Replicas && tc.Spec.TiKV.Replicas == oldTc.Spec.TiKV.Replicas {
		return nil
	}
	newTc := tc.DeepCopy()
	_, err := am.tcControl.UpdateTidbCluster(newTc, &newTc.Status, &oldTc.Status)
	if err != nil {
		return err
	}
	reason := fmt.Sprintf("Successful %s", strings.Title("auto-scaling"))
	msg := ""
	if tc.Spec.TiDB.Replicas != oldTc.Spec.TiDB.Replicas {
		msg = fmt.Sprintf("%s auto-scaling tidb from %d to %d", msg, oldTc.Spec.TiDB.Replicas, tc.Spec.TiDB.Replicas)
	}
	if tc.Spec.TiKV.Replicas != oldTc.Spec.TiKV.Replicas {
		msg = fmt.Sprintf("%s auto-scaling tikv from %d to %d", msg, oldTc.Spec.TiKV.Replicas, tc.Spec.TiKV.Replicas)
	}
	am.recorder.Event(tac, corev1.EventTypeNormal, reason, msg)
	return nil
}

func (am *autoScalerManager) updateAutoScaling(oldTc *v1alpha1.TidbCluster,
	tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Annotations == nil {
		tac.Annotations = map[string]string{}
	}
	now := time.Now()
	tac.Annotations[label.AnnLastSyncingTimestamp] = fmt.Sprintf("%d", now.Unix())
	if tac.Spec.TiKV != nil {
		if oldTc.Status.TiKV.StatefulSet != nil {
			tac.Status.TiKV.CurrentReplicas = oldTc.Status.TiKV.StatefulSet.CurrentReplicas
		}
		tac.Status.TiKV.LastAutoScalingTimestamp = &metav1.Time{Time: now}
	} else {
		tac.Status.TiKV = nil
	}

	if tac.Spec.TiDB != nil {
		if oldTc.Status.TiDB.StatefulSet != nil {
			tac.Status.TiDB.CurrentReplicas = oldTc.Status.TiDB.StatefulSet.CurrentReplicas
		}
		tac.Status.TiDB.LastAutoScalingTimestamp = &metav1.Time{Time: now}
	} else {
		tac.Status.TiDB = nil
	}
	return am.updateTidbClusterAutoScaler(tac)
}

func (am *autoScalerManager) updateTidbClusterAutoScaler(tac *v1alpha1.TidbClusterAutoScaler) error {

	ns := tac.GetNamespace()
	tacName := tac.GetName()
	oldTac := tac.DeepCopy()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		_, updateErr = am.cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Update(tac)
		if updateErr == nil {
			klog.Infof("TidbClusterAutoScaler: [%s/%s] updated successfully", ns, tacName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbClusterAutoScaler: [%s/%s], error: %v", ns, tacName, updateErr)
		if updated, err := am.taLister.TidbClusterAutoScalers(ns).Get(tacName); err == nil {
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

// checkAutoScalerRef will first check whether the target tidbcluster's auto-scaler reference have been occupied.
// If it has been, and the reference scaler is the current auto-scaler itself, the auto-scaler would be permitted,
// otherwise the auto-scaling would be forbidden.
// If the target tidbcluster's auto-scaler reference is empty, then the auto-scaler will try to patch itself to the
// references, and if the patching is success, the auto-scaling would discard the current syncing and wait for the next
// syncing round.
func (am *autoScalerManager) checkAutoScalerRef(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) (bool, error) {
	if tc.Status.AutoScaler != nil {
		if tc.Status.AutoScaler.Name == tac.Name && tc.Status.AutoScaler.Namespace == tac.Namespace {
			return true, nil
		}
		msg := fmt.Sprintf("tac[%s/%s]'s target tc[%s/%s] already controlled by another auto-scaler", tac.Namespace, tac.Name, tc.Namespace, tc.Name)
		klog.Info(msg)
		return false, nil
	}
	klog.Infof("tac[%s/%s]'s tc[%s/%s] start to occupy the auto-scaler ref", tac.Namespace, tac.Name, tc.Namespace, tc.Name)
	err := am.patchAutoScalerRef(tc, tac)
	return true, err
}

func (am *autoScalerManager) patchAutoScalerRef(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	mergePatch, err := json.Marshal(map[string]interface{}{
		"status": map[string]interface{}{
			"auto-scaler": map[string]interface{}{
				"name":      tac.Name,
				"namespace": tac.Namespace,
			},
		},
	})
	if err != nil {
		klog.Error(err)
		return err
	}
	_, err = am.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Patch(tc.Name, types.MergePatchType, mergePatch)
	if err != nil {
		klog.Error(err)
		return err
	}
	msg := fmt.Sprintf("tac[%s/%s] patch itself to tc[%s/%s] auto-scaler ref success, do auto-scaling in next round", tac.Namespace, tac.Name, tc.Namespace, tc.Name)
	return controller.RequeueErrorf(msg)
}
