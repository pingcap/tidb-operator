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
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type ManifestFetcher interface {
	ListByTC(tc *pingcapv1alpha1.TidbCluster) (objects []runtime.Object, err error)
}

type TiDBDashboardFetcher struct {
	lister listers.TidbDashboardLister
}

func (f *TiDBDashboardFetcher) ListByTC(tc *pingcapv1alpha1.TidbCluster) (objects []runtime.Object, err error) {
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

func (f *TiDBMonitorFetcher) ListByTC(tc *pingcapv1alpha1.TidbCluster) (objects []runtime.Object, err error) {
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

func (f *TiDBClusterAutoScalerFetcher) ListByTC(tc *pingcapv1alpha1.TidbCluster) (objects []runtime.Object, err error) {
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
			objects = append(objects, autoScaler)
		}
	}
	return
}

type TiDBInitializerFetcher struct {
	lister listers.TidbInitializerLister
}

func (f *TiDBInitializerFetcher) ListByTC(tc *pingcapv1alpha1.TidbCluster) (objects []runtime.Object, err error) {
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
			objects = append(objects, initializer)
		}
	}
	return
}

type TiDBNgMonitoringFetcher struct {
	lister listers.TidbNGMonitoringLister
}

func (f *TiDBNgMonitoringFetcher) ListByTC(tc *pingcapv1alpha1.TidbCluster) (objects []runtime.Object, err error) {
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
				objects = append(objects, monitoring)
				break
			}
		}
	}
	return
}
