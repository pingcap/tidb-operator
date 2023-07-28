// Copyright 2023 PingCAP, Inc.
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

package backup

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

type ManifestFetcher interface {
	ListByTC(tc *v1alpha1.TidbCluster) (objects []runtime.Object, err error)
}

type TiDBDashboardFetcher struct {
	lister listers.TidbDashboardLister
}

func NewTiDBDashboardFetcher(lister listers.TidbDashboardLister) *TiDBDashboardFetcher {
	return &TiDBDashboardFetcher{
		lister: lister,
	}
}

func (f *TiDBDashboardFetcher) ListByTC(tc *v1alpha1.TidbCluster) (objects []runtime.Object, err error) {
	emptySelector := labels.NewSelector()
	dashboards, err := f.lister.List(emptySelector)
	if err != nil {
		return nil, err
	}

	for _, dashboard := range dashboards {
		for _, cluster := range dashboard.Spec.Clusters {
			name := cluster.Name
			namespace := cluster.Namespace
			if namespace == "" {
				namespace = dashboard.Namespace
			}
			if name == tc.Name && namespace == tc.Namespace {
				klog.Infof("TiDBDashboard %s/%s matches tc %s/%s",
					dashboard.Namespace, dashboard.Name, tc.Namespace, tc.Name)
				objects = append(objects, dashboard)
				break
			}
		}
	}
	return
}

type TiDBMonitorFetcher struct {
	lister listers.TidbMonitorLister
}

func NewTiDBMonitorFetcher(lister listers.TidbMonitorLister) *TiDBMonitorFetcher {
	return &TiDBMonitorFetcher{
		lister: lister,
	}
}

func (f *TiDBMonitorFetcher) ListByTC(tc *v1alpha1.TidbCluster) (objects []runtime.Object, err error) {
	emptySelector := labels.NewSelector()
	monitors, err := f.lister.List(emptySelector)
	if err != nil {
		return nil, err
	}

	for _, monitor := range monitors {
		for _, cluster := range monitor.Spec.Clusters {
			name := cluster.Name
			namespace := cluster.Namespace
			if namespace == "" {
				namespace = monitor.Namespace
			}
			if name == tc.Name && namespace == tc.Namespace {
				klog.Infof("TidbMonitor %s/%s matches tc %s/%s",
					monitor.Namespace, monitor.Name, tc.Namespace, tc.Name)
				objects = append(objects, monitor)
				break
			}
		}
	}
	return
}

type TiDBClusterAutoScalerFetcher struct {
	lister listers.TidbClusterAutoScalerLister
}

func NewTiDBClusterAutoScalerFetcher(lister listers.TidbClusterAutoScalerLister) *TiDBClusterAutoScalerFetcher {
	return &TiDBClusterAutoScalerFetcher{
		lister: lister,
	}
}

func (f *TiDBClusterAutoScalerFetcher) ListByTC(tc *v1alpha1.TidbCluster) (objects []runtime.Object, err error) {
	emptySelector := labels.NewSelector()
	autoScalers, err := f.lister.List(emptySelector)
	if err != nil {
		return nil, err
	}

	for _, autoScaler := range autoScalers {
		name := autoScaler.Spec.Cluster.Name
		namespace := autoScaler.Spec.Cluster.Namespace
		if namespace == "" {
			namespace = autoScaler.Namespace
		}
		if name == tc.Name && namespace == tc.Namespace {
			klog.Infof("TiDBClusterAutoScaler %s/%s matches tc %s/%s",
				autoScaler.Namespace, autoScaler.Name, tc.Namespace, tc.Name)
			objects = append(objects, autoScaler)
		}
	}
	return
}

type TiDBInitializerFetcher struct {
	lister listers.TidbInitializerLister
}

func NewTiDBInitializerFetcher(lister listers.TidbInitializerLister) *TiDBInitializerFetcher {
	return &TiDBInitializerFetcher{
		lister: lister,
	}
}

func (f *TiDBInitializerFetcher) ListByTC(tc *v1alpha1.TidbCluster) (objects []runtime.Object, err error) {
	emptySelector := labels.NewSelector()
	initializers, err := f.lister.List(emptySelector)
	if err != nil {
		return nil, err
	}

	for _, initializer := range initializers {
		name := initializer.Spec.Clusters.Name
		namespace := initializer.Spec.Clusters.Namespace
		if namespace == "" {
			namespace = initializer.Namespace
		}
		if name == tc.Name && namespace == tc.Namespace {
			klog.Infof("TiDBInitializer %s/%s matches tc %s/%s",
				initializer.Namespace, initializer.Name, tc.Namespace, tc.Name)
			objects = append(objects, initializer)
		}
	}
	return
}

type TiDBNgMonitoringFetcher struct {
	lister listers.TidbNGMonitoringLister
}

func NewTiDBNgMonitoringFetcher(lister listers.TidbNGMonitoringLister) *TiDBNgMonitoringFetcher {
	return &TiDBNgMonitoringFetcher{
		lister: lister,
	}
}

func (f *TiDBNgMonitoringFetcher) ListByTC(tc *v1alpha1.TidbCluster) (objects []runtime.Object, err error) {
	emptySelector := labels.NewSelector()
	monitorings, err := f.lister.List(emptySelector)
	if err != nil {
		return nil, err
	}

	for _, monitoring := range monitorings {
		for _, cluster := range monitoring.Spec.Clusters {
			name := cluster.Name
			namespace := cluster.Namespace
			if namespace == "" {
				namespace = monitoring.Namespace
			}
			if name == tc.Name && namespace == tc.Namespace {
				klog.Infof("TidbNGMonitoring %s/%s matches tc %s/%s",
					monitoring.Namespace, monitoring.Name, tc.Namespace, tc.Name)
				objects = append(objects, monitoring)
				break
			}
		}
	}
	return
}
