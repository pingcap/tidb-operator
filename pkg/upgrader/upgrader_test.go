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

package upgrader

import (
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	asappsv1 "github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1"
	asclientsetfake "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	versionedfake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeleteSlotAnns(t *testing.T) {
	tests := []struct {
		name string
		tc   *v1alpha1.TidbCluster
		want map[string]string
	}{
		{
			name: "tc nil",
			tc:   nil,
			want: map[string]string{},
		},
		{
			name: "tc anns nil",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			want: map[string]string{},
		},
		{
			name: "tc anns empty",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			want: map[string]string{},
		},
		{
			name: "tc anns no delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
			},
			want: map[string]string{},
		},
		{
			name: "tc anns has delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
			},
			want: map[string]string{
				label.AnnTiDBDeleteSlots: "[1,2]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deleteSlotAnns(tt.tc)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}

func TestUpgrade(t *testing.T) {
	tests := []struct {
		name                     string
		tidbClusters             []v1alpha1.TidbCluster
		statefulsets             []appsv1.StatefulSet
		feature                  string
		ns                       string
		wantAdvancedStatefulsets []asappsv1.StatefulSet
		wantStatefulsets         []appsv1.StatefulSet
		wantErr                  bool
	}{
		{
			name: "basic",
			statefulsets: []appsv1.StatefulSet{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts1",
						Namespace: "sts",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts2",
						Namespace: "sts",
					},
				},
			},
			feature: "AdvancedStatefulSet=true",
			ns:      metav1.NamespaceAll,
			wantErr: false,
			wantAdvancedStatefulsets: []asappsv1.StatefulSet{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps.pingcap.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts1",
						Namespace: "sts",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps.pingcap.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts2",
						Namespace: "sts",
					},
				},
			},
			wantStatefulsets: nil,
		},
		{
			name: "other namespaces should not be affected if not cluster scoped",
			statefulsets: []appsv1.StatefulSet{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts1",
						Namespace: "sts",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts2",
						Namespace: "sts",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other2",
						Namespace: "other",
					},
				},
			},
			feature: "AdvancedStatefulSet=true",
			ns:      "sts",
			wantErr: false,
			wantAdvancedStatefulsets: []asappsv1.StatefulSet{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps.pingcap.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts1",
						Namespace: "sts",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps.pingcap.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts2",
						Namespace: "sts",
					},
				},
			},
			wantStatefulsets: []appsv1.StatefulSet{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other2",
						Namespace: "other",
					},
				},
			},
		},
		{
			name: "should not upgrade if tc has delete slot annotations",
			tidbClusters: []v1alpha1.TidbCluster{
				{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							label.AnnTiDBDeleteSlots: "[1,2]",
						},
					},
				},
			},
			statefulsets: []appsv1.StatefulSet{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts1",
						Namespace: "sts",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts2",
						Namespace: "sts",
					},
				},
			},
			feature:                  "AdvancedStatefulSet=true",
			ns:                       metav1.NamespaceAll,
			wantErr:                  true,
			wantAdvancedStatefulsets: nil,
			wantStatefulsets: []appsv1.StatefulSet{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts1",
						Namespace: "sts",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sts2",
						Namespace: "sts",
					},
				},
			},
		},
	}

	// these tests must run serially, because we share features.DefaultFeatureGate
	for _, tt := range tests {
		t.Logf("Testing %s", tt.name)

		features.DefaultFeatureGate.Set(tt.feature)

		var err error
		kubeCli := fake.NewSimpleClientset()
		asCli := asclientsetfake.NewSimpleClientset()
		cli := versionedfake.NewSimpleClientset()

		for _, tc := range tt.tidbClusters {
			_, err = cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(&tc)
			if err != nil {
				t.Fatal(err)
			}
		}

		for _, sts := range tt.statefulsets {
			_, err = kubeCli.AppsV1().StatefulSets(sts.Namespace).Create(&sts)
			if err != nil {
				t.Fatal(err)
			}
		}

		operatorUpgrader := NewUpgrader(kubeCli, cli, asCli, tt.ns)
		err = operatorUpgrader.Upgrade()
		if tt.wantErr {
			if err == nil {
				t.Errorf("expected err, got %v", err)
			}
		} else {
			if err != nil {
				t.Errorf("expected no err, got %v", err)
			}
		}

		gotAdvancedStatfulSetsList, err := asCli.AppsV1().StatefulSets(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}

		sort.Sort(sortAdvancedStatefulsetByNamespaceName(gotAdvancedStatfulSetsList.Items))
		if diff := cmp.Diff(tt.wantAdvancedStatefulsets, gotAdvancedStatfulSetsList.Items); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}

		gotStatfulSetsList, err := kubeCli.AppsV1().StatefulSets(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}

		sort.Sort(sortStatefulSetByNamespace(gotStatfulSetsList.Items))
		if diff := cmp.Diff(tt.wantStatefulsets, gotStatfulSetsList.Items); diff != "" {
			t.Errorf("unexpected (-want, +got): %s", diff)
		}
	}
}

type sortAdvancedStatefulsetByNamespaceName []asappsv1.StatefulSet

func (o sortAdvancedStatefulsetByNamespaceName) Len() int      { return len(o) }
func (o sortAdvancedStatefulsetByNamespaceName) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o sortAdvancedStatefulsetByNamespaceName) Less(i, j int) bool {
	return fmt.Sprintf("%s/%s", o[i].GetNamespace(), o[i].GetName()) < fmt.Sprintf("%s/%s", o[j].GetNamespace(), o[j].GetName())
}

type sortStatefulSetByNamespace []appsv1.StatefulSet

func (o sortStatefulSetByNamespace) Len() int      { return len(o) }
func (o sortStatefulSetByNamespace) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o sortStatefulSetByNamespace) Less(i, j int) bool {
	return fmt.Sprintf("%s/%s", o[i].GetNamespace(), o[i].GetName()) < fmt.Sprintf("%s/%s", o[j].GetNamespace(), o[j].GetName())
}
