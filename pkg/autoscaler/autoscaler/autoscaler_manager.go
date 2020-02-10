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
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type autoScalerManager struct {
	tcLister v1alpha1listers.TidbClusterLister
	recorder record.EventRecorder
}

func NewAutoScalerManager(
	informerFactory informers.SharedInformerFactory,
	recorder record.EventRecorder) *autoScalerManager {
	return &autoScalerManager{
		tcLister: informerFactory.Pingcap().V1alpha1().TidbClusters().Lister(),
		recorder: recorder,
	}
}

func (am *autoScalerManager) Sync(tac *v1alpha1.TidbClusterAutoScaler) error {
	if tac.DeletionTimestamp != nil {
		return nil
	}
	tcName := tac.Spec.Cluster.Name
	tcNamespace := tac.Spec.Cluster.Namespace
	tc, err := am.tcLister.TidbClusters(tcNamespace).Get(tcName)
	if err != nil {
		return err
	}
	if tac.Spec.MetricsUrl == nil {
		return fmt.Errorf("tidbclusterAutoScaler[%s/%s]' metrics url should be defined currently", tac.Namespace, tac.Name)
	}

	if err := am.syncTiKV(tc, tac); err != nil {
		return err
	}

	if err := am.syncTiKV(tc, tac); err != nil {
		return err
	}

	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return nil
}

func (am *autoScalerManager) syncTiKV(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	return nil
}

func (am *autoScalerManager) syncTiDB(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler) error {
	return nil
}
