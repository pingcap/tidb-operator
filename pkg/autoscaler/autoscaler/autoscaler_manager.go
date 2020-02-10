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
	"context"
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	promClient "github.com/prometheus/client_golang/api"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
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
	client, err := promClient.NewClient(promClient.Config{Address: *tac.Spec.MetricsUrl})
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := am.syncTiKV(tc, tac, client, ctx); err != nil {
		return err
	}

	if err := am.syncTiKV(tc, tac, client, ctx); err != nil {
		return err
	}

	klog.Infof("tc[%s/%s]'s tac[%s/%s] synced", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
	return nil
}

//sum(rate(tikv_thread_cpu_seconds_total{cluster="tidb"}[1m])) by (instance)
//rate(process_cpu_seconds_total{cluster="tidb",job="tikv"}[1m]) ??
//sum(rate(tikv_grpc_msg_duration_seconds_count{cluster="tidb", type!="kv_gc"}[1m])) by (instance)
func (am *autoScalerManager) syncTiKV(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, client promClient.Client, ctx context.Context) error {
	if tac.Spec.TiKV == nil {
		return nil
	}
	// tikv is under updated, refuse to auto-scaling
	if tc.Status.TiKV.StatefulSet.CurrentRevision != tc.Status.TiKV.StatefulSet.UpdateRevision {
		return nil
	}
	return nil
}

//rate(process_cpu_seconds_total{cluster="tidb",job="tidb"}[1m])
func (am *autoScalerManager) syncTiDB(tc *v1alpha1.TidbCluster, tac *v1alpha1.TidbClusterAutoScaler, client promClient.Client, ctx context.Context) error {
	if tac.Spec.TiDB == nil {
		return nil
	}

	// tidb is under updated, refuse to auto-scaling
	if tc.Status.TiDB.StatefulSet.CurrentRevision != tc.Status.TiDB.StatefulSet.UpdateRevision {
		return nil
	}
	for _, metric := range tac.Spec.TiDB.Metrics {
		// TIDB auto-scaler only support CPU AverageUtilization metrics
		if metric.Type == autoscalingv2beta2.ResourceMetricSourceType &&
			metric.Resource != nil &&
			metric.Resource.Name == corev1.ResourceCPU &&
			metric.Resource.Target.AverageUtilization != nil {
			
		} else {
			return fmt.Errorf("tidbclusterAutoScaler[%s/%s] only support ")
		}
	}
	return nil
}
