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

package member

import (
	"context"
	"reflect"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	versionedfake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPodRestarterSync(t *testing.T) {
	tests := []struct {
		name          string
		tc            *v1alpha1.TidbCluster
		pods          []*v1.Pod
		wantRestarted []bool
		wantErr       error
	}{
		{
			name: "no pods",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{},
			},
		},
		{
			name: "pods do no deleting annotation",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
			},
			pods: []*v1.Pod{
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: v1.NamespaceDefault,
						Name:      "pod",
						Labels: map[string]string{
							label.NameLabelKey:      "tidb-cluster",
							label.ManagedByLabelKey: label.TiDBOperator,
							label.InstanceLabelKey:  "tc",
						},
					},
				},
			},
			wantRestarted: []bool{false},
		},
		{
			name: "one pod is restarted",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: v1.NamespaceDefault,
					Name:      "tc",
				},
			},
			pods: []*v1.Pod{
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: v1.NamespaceDefault,
						Name:      "pod-1",
						Labels: map[string]string{
							label.NameLabelKey:      "tidb-cluster",
							label.ManagedByLabelKey: label.TiDBOperator,
							label.InstanceLabelKey:  "tc",
						},
						Annotations: map[string]string{
							label.AnnPodDeferDeleting: "yes",
						},
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: v1.NamespaceDefault,
						Name:      "pod-2",
						Labels: map[string]string{
							label.NameLabelKey:      "tidb-cluster",
							label.ManagedByLabelKey: label.TiDBOperator,
							label.InstanceLabelKey:  "tc",
						},
					},
				},
			},
			wantRestarted: []bool{true, false},
			wantErr:       controller.RequeueErrorf("tc[default/tc] is under restarting"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cli := versionedfake.NewSimpleClientset()
			kubeCli := fake.NewSimpleClientset()
			if tt.tc != nil {
				cli.PingcapV1alpha1().TidbClusters(tt.tc.Namespace).Create(tt.tc)
			}
			for _, pod := range tt.pods {
				kubeCli.CoreV1().Pods(pod.Namespace).Create(pod)
			}

			informerFactory := informers.NewSharedInformerFactory(kubeCli, 0)
			restarter := NewPodRestarter(kubeCli, informerFactory.Core().V1().Pods().Lister())

			informerFactory.Start(ctx.Done())
			informerFactory.WaitForCacheSync(ctx.Done())

			err := restarter.Sync(tt.tc)

			if !reflect.DeepEqual(tt.wantErr, err) {
				t.Errorf("want %v, got %v", tt.wantErr, err)
			}

			for i, pod := range tt.pods {
				wantRestarted := tt.wantRestarted[i]
				_, err := kubeCli.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
				if wantRestarted {
					if !apierrors.IsNotFound(err) {
						t.Errorf("expects pod %q does not exist, got %v", pod.Name, err)
					}
				} else {
					if err != nil {
						t.Errorf("expects err to be nil, got %v", err)
					}
				}
			}
		})
	}
}
