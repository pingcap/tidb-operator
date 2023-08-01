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
	"testing"

	"github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers_v1alpha1 "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type fakeIndexer struct {
	cache.Indexer
	listResults []interface{}
}

func (f *fakeIndexer) List() []interface{} {
	return f.listResults
}

func TestTiDBDashboardFetcher(t *testing.T) {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "tc-1",
		},
	}
	dashboards := []*v1alpha1.TidbDashboard{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-1",
			},
			Spec: v1alpha1.TidbDashboardSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name: "tc-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-2",
				Name:      "d-2",
			},
			Spec: v1alpha1.TidbDashboardSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name: "tc-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-3",
			},
			Spec: v1alpha1.TidbDashboardSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-1",
						Namespace: "ns-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-4",
			},
			Spec: v1alpha1.TidbDashboardSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-1",
						Namespace: "ns-2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-5",
			},
			Spec: v1alpha1.TidbDashboardSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-2",
						Namespace: "ns-1",
					},
				},
			},
		},
	}
	expectedResults := []*v1alpha1.TidbDashboard{dashboards[0], dashboards[2]}
	g := gomega.NewGomegaWithT(t)

	listResults := make([]interface{}, 0, len(dashboards))
	for _, d := range dashboards {
		listResults = append(listResults, d)
	}
	lister := listers_v1alpha1.NewTidbDashboardLister(&fakeIndexer{
		listResults: listResults,
	})
	fetcher := NewTiDBDashboardFetcher(lister)
	results, err := fetcher.ListByTC(tc)
	g.Expect(err, gomega.BeNil())
	g.Expect(results, gomega.Equal(expectedResults))
}

func TestTiDBMonitorFetcher(t *testing.T) {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "tc-1",
		},
	}
	monitors := []*v1alpha1.TidbMonitor{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-1",
			},
			Spec: v1alpha1.TidbMonitorSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name: "tc-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-2",
				Name:      "d-2",
			},
			Spec: v1alpha1.TidbMonitorSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name: "tc-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-3",
			},
			Spec: v1alpha1.TidbMonitorSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-1",
						Namespace: "ns-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-4",
			},
			Spec: v1alpha1.TidbMonitorSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-1",
						Namespace: "ns-2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-5",
			},
			Spec: v1alpha1.TidbMonitorSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-2",
						Namespace: "ns-1",
					},
				},
			},
		},
	}
	expectedResults := []*v1alpha1.TidbMonitor{monitors[0], monitors[2]}
	g := gomega.NewGomegaWithT(t)

	listResults := make([]interface{}, 0, len(monitors))
	for _, m := range monitors {
		listResults = append(listResults, m)
	}
	lister := listers_v1alpha1.NewTidbMonitorLister(&fakeIndexer{
		listResults: listResults,
	})
	fetcher := NewTiDBMonitorFetcher(lister)
	results, err := fetcher.ListByTC(tc)
	g.Expect(err, gomega.BeNil())
	g.Expect(results, gomega.Equal(expectedResults))
}

func TestTiDBClusterAutoScalerFetcher(t *testing.T) {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "tc-1",
		},
	}
	autoScalers := []*v1alpha1.TidbClusterAutoScaler{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-1",
			},
			Spec: v1alpha1.TidbClusterAutoScalerSpec{
				Cluster: v1alpha1.TidbClusterRef{
					Name: "tc-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-2",
				Name:      "d-2",
			},
			Spec: v1alpha1.TidbClusterAutoScalerSpec{
				Cluster: v1alpha1.TidbClusterRef{
					Name: "tc-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-3",
			},
			Spec: v1alpha1.TidbClusterAutoScalerSpec{
				Cluster: v1alpha1.TidbClusterRef{
					Name:      "tc-1",
					Namespace: "ns-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-4",
			},
			Spec: v1alpha1.TidbClusterAutoScalerSpec{
				Cluster: v1alpha1.TidbClusterRef{
					Name:      "tc-1",
					Namespace: "ns-2",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-5",
			},
			Spec: v1alpha1.TidbClusterAutoScalerSpec{
				Cluster: v1alpha1.TidbClusterRef{
					Name:      "tc-2",
					Namespace: "ns-1",
				},
			},
		},
	}
	expectedResults := []*v1alpha1.TidbClusterAutoScaler{autoScalers[0], autoScalers[2]}
	g := gomega.NewGomegaWithT(t)

	listResults := make([]interface{}, 0, len(autoScalers))
	for _, a := range autoScalers {
		listResults = append(listResults, a)
	}
	lister := listers_v1alpha1.NewTidbClusterAutoScalerLister(&fakeIndexer{
		listResults: listResults,
	})
	fetcher := NewTiDBClusterAutoScalerFetcher(lister)
	results, err := fetcher.ListByTC(tc)
	g.Expect(err, gomega.BeNil())
	g.Expect(results, gomega.Equal(expectedResults))
}

func TestTiDBInitializerFetcher(t *testing.T) {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "tc-1",
		},
	}
	initializers := []*v1alpha1.TidbInitializer{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-1",
			},
			Spec: v1alpha1.TidbInitializerSpec{
				Clusters: v1alpha1.TidbClusterRef{
					Name: "tc-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-2",
				Name:      "d-2",
			},
			Spec: v1alpha1.TidbInitializerSpec{
				Clusters: v1alpha1.TidbClusterRef{
					Name: "tc-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-3",
			},
			Spec: v1alpha1.TidbInitializerSpec{
				Clusters: v1alpha1.TidbClusterRef{
					Name:      "tc-1",
					Namespace: "ns-1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-4",
			},
			Spec: v1alpha1.TidbInitializerSpec{
				Clusters: v1alpha1.TidbClusterRef{
					Name:      "tc-1",
					Namespace: "ns-2",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-5",
			},
			Spec: v1alpha1.TidbInitializerSpec{
				Clusters: v1alpha1.TidbClusterRef{
					Name:      "tc-2",
					Namespace: "ns-1",
				},
			},
		},
	}
	expectedResults := []*v1alpha1.TidbInitializer{initializers[0], initializers[2]}
	g := gomega.NewGomegaWithT(t)

	listResults := make([]interface{}, 0, len(initializers))
	for _, a := range initializers {
		listResults = append(listResults, a)
	}
	lister := listers_v1alpha1.NewTidbInitializerLister(&fakeIndexer{
		listResults: listResults,
	})
	fetcher := NewTiDBInitializerFetcher(lister)
	results, err := fetcher.ListByTC(tc)
	g.Expect(err, gomega.BeNil())
	g.Expect(results, gomega.Equal(expectedResults))
}

func TestTiDBNgMonitoringFetcher(t *testing.T) {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns-1",
			Name:      "tc-1",
		},
	}
	monitorings := []*v1alpha1.TidbNGMonitoring{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-1",
			},
			Spec: v1alpha1.TidbNGMonitoringSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name: "tc-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-2",
				Name:      "d-2",
			},
			Spec: v1alpha1.TidbNGMonitoringSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name: "tc-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-3",
			},
			Spec: v1alpha1.TidbNGMonitoringSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-1",
						Namespace: "ns-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-4",
			},
			Spec: v1alpha1.TidbNGMonitoringSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-1",
						Namespace: "ns-2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "d-5",
			},
			Spec: v1alpha1.TidbNGMonitoringSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Name:      "tc-2",
						Namespace: "ns-1",
					},
				},
			},
		},
	}
	expectedResults := []*v1alpha1.TidbNGMonitoring{monitorings[0], monitorings[2]}
	g := gomega.NewGomegaWithT(t)

	listResults := make([]interface{}, 0, len(monitorings))
	for _, m := range monitorings {
		listResults = append(listResults, m)
	}
	lister := listers_v1alpha1.NewTidbNGMonitoringLister(&fakeIndexer{
		listResults: listResults,
	})
	fetcher := NewTiDBNgMonitoringFetcher(lister)
	results, err := fetcher.ListByTC(tc)
	g.Expect(err, gomega.BeNil())
	g.Expect(results, gomega.Equal(expectedResults))
}
