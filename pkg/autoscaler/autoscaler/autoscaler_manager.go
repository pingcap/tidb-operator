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
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	promClient "github.com/prometheus/client_golang/api"
	"k8s.io/apimachinery/pkg/api/errors"
	kubeinformers "k8s.io/client-go/informers"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type autoScalerManager struct {
	tcLister  v1alpha1listers.TidbClusterLister
	stsLister appslisters.StatefulSetLister
	recorder  record.EventRecorder
}

func NewAutoScalerManager(
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) *autoScalerManager {
	return &autoScalerManager{
		tcLister:  informerFactory.Pingcap().V1alpha1().TidbClusters().Lister(),
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

	tcNamespace := tac.Spec.Cluster.Namespace
	tc, err := am.tcLister.TidbClusters(tcNamespace).Get(tcName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Target TidbCluster Ref is deleted, empty the auto-scaling status
			emptyAutoScalingCountAnn(tac, v1alpha1.TiDBMemberType)
			emptyAutoScalingCountAnn(tac, v1alpha1.TiKVMemberType)
		}
		return err
	}
	checkAndUpdateTacAnn(tac)
	oldTCSpec := tc.Spec.DeepCopy()
	if err := am.syncAutoScaling(tc, tac); err != nil {
		return err
	}
	if err := am.syncTidbClusterReplicas(tc, oldTCSpec); err != nil {
		return err
	}
	return am.syncAutoScalingStatus(tc, oldTCSpec, tac)
}

func (am *autoScalerManager) syncAutoScaling(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.Spec.MetricsUrl == nil {
		return fmt.Errorf("tidbclusterAutoScaler[%s/%s]' metrics url should be defined explicitly", tac.Namespace, tac.Name)
	}
	client, err := promClient.NewClient(promClient.Config{Address: *tac.Spec.MetricsUrl})
	if err != nil {
		return err
	}
	defaultTAC(tac)
	if err := am.syncTiKV(tc, tac, client); err != nil {
		return err
	}
	if err := am.syncTiDB(tc, tac, client); err != nil {
		return err
	}
	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return nil
}

//TODO: sync TidbCluster.Spec.Replicas
func (am *autoScalerManager) syncTidbClusterReplicas(tc *v1alpha1.TidbCluster, oldTCSpec *v1alpha1.TidbClusterSpec) error {
	return nil
}

//TODO: sync tac status
func (am *autoScalerManager) syncAutoScalingStatus(tc *v1alpha1.TidbCluster, oldTc *v1alpha1.TidbClusterSpec,
	tac *v1alpha1.TidbClusterAutoScaler) error {
	return nil
}
