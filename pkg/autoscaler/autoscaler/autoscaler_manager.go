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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	promClient "github.com/prometheus/client_golang/api"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

type autoScalerManager struct {
	cli       versioned.Interface
	tcControl controller.TidbClusterControlInterface
	taLister  v1alpha1listers.TidbClusterAutoScalerLister
	stsLister appslisters.StatefulSetLister
	recorder  record.EventRecorder
}

func NewAutoScalerManager(
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) *autoScalerManager {
	tcLister := informerFactory.Pingcap().V1alpha1().TidbClusters().Lister()
	stsLister := kubeInformerFactory.Apps().V1().StatefulSets().Lister()
	return &autoScalerManager{
		cli:       cli,
		tcControl: controller.NewRealTidbClusterControl(cli, tcLister, recorder),
		taLister:  informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers().Lister(),
		stsLister: stsLister,
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

	tc, err := am.cli.PingcapV1alpha1().TidbClusters(tac.Spec.Cluster.Namespace).Get(tcName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Target TidbCluster Ref is deleted, empty the auto-scaling status
			resetAutoScalingAnn(tac)
			return nil
		}
		return err
	}
	checkAndUpdateTacAnn(tac)
	oldTc := tc.DeepCopy()
	if err := am.syncAutoScaling(tc, tac); err != nil {
		return err
	}
	if err := am.syncTidbClusterReplicas(tc, oldTc); err != nil {
		return err
	}
	return am.syncAutoScalingStatus(tc, oldTc, tac)
}

func (am *autoScalerManager) syncAutoScaling(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Spec.MetricsUrl == nil {
		return fmt.Errorf("tidbclusterAutoScaler[%s/%s]' metrics url should be defined explicitly", tac.Namespace, tac.Name)
	}
	c, err := promClient.NewClient(promClient.Config{Address: *tac.Spec.MetricsUrl})
	if err != nil {
		return err
	}
	defaultTAC(tac)
	oldTikvReplicas := tc.Spec.TiKV.Replicas
	if err := am.syncTiKV(tc, tac, c); err != nil {
		tc.Spec.TiKV.Replicas = oldTikvReplicas
		klog.Errorf("tac[%s/%s] tikv sync failed, continue to sync next, err:%v", tac.Namespace, tac.Name, err)
	}
	oldTidbReplicas := tc.Spec.TiDB.Replicas
	if err := am.syncTiDB(tc, tac, c); err != nil {
		tc.Spec.TiDB.Replicas = oldTidbReplicas
		klog.Errorf("tac[%s/%s] tidb sync failed, continue to sync next, err:%v", tac.Namespace, tac.Name, err)
	}
	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return nil
}

func (am *autoScalerManager) syncTidbClusterReplicas(tc *v1alpha1.TidbCluster, oldTc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiDB.Replicas == oldTc.Spec.TiDB.Replicas && tc.Spec.TiKV.Replicas == oldTc.Spec.TiKV.Replicas {
		return nil
	}
	newTc := tc.DeepCopy()
	_, err := am.tcControl.UpdateTidbCluster(newTc, &newTc.Status, &oldTc.Status)
	if err != nil {
		return err
	}
	return nil
}

//TODO: sync tac status
func (am *autoScalerManager) syncAutoScalingStatus(tc *v1alpha1.TidbCluster, oldTc *v1alpha1.TidbCluster,
	tac *v1alpha1.TidbClusterAutoScaler) error {
	return am.updateTidbClusterAutoScaler(tac)
}

func (am *autoScalerManager) updateTidbClusterAutoScaler(tac *v1alpha1.TidbClusterAutoScaler) error {

	ns := tac.GetNamespace()
	tacName := tac.GetName()
	oldTac := tac.DeepCopy()

	// don't wait due to limited number of clients, but backoff after the default number of steps
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		_, updateErr = am.cli.PingcapV1alpha1().TidbClusterAutoScalers(ns).Update(tac)
		if updateErr == nil {
			klog.Infof("TidbClusterAutoScaler: [%s/%s] updated successfully", ns, tacName)
			return nil
		}
		klog.Errorf("failed to update TidbClusterAutoScaler: [%s/%s], error: %v", ns, tacName, updateErr)
		if updated, err := am.taLister.TidbClusterAutoScalers(ns).Get(tacName); err == nil {
			// make a copy so we don't mutate the shared cache
			tac = updated.DeepCopy()
			tac.Annotations = oldTac.Annotations
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbClusterAutoScaler %s/%s from lister: %v", ns, tacName, err))
		}
		return updateErr
	})
}
