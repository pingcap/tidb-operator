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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestSyncAutoScalerRef(t *testing.T) {
	g := NewGomegaWithT(t)
	testcases := []struct {
		name              string
		haveRef           bool
		autoScalerExisted bool
		correctRef        bool
		expectedStatusRef *v1alpha1.TidbClusterAutoScalerRef
	}{
		{
			name:              "empty Reference",
			haveRef:           false,
			autoScalerExisted: false,
			correctRef:        false,
			expectedStatusRef: nil,
		},
		{
			name:              "normal",
			haveRef:           true,
			autoScalerExisted: true,
			correctRef:        true,
			expectedStatusRef: &v1alpha1.TidbClusterAutoScalerRef{
				Name:      "auto-scaler",
				Namespace: "default",
			},
		},
		{
			name:              "target auto-scaler not existed",
			haveRef:           true,
			autoScalerExisted: false,
			correctRef:        false,
			expectedStatusRef: nil,
		},
		{
			name:              "target auto-scaler have changed the cluster target",
			haveRef:           true,
			autoScalerExisted: true,
			correctRef:        false,
			expectedStatusRef: nil,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			tsm, _, _, scalerInder := newFakeTidbClusterStatusManager()
			tc := newTidbCluster()
			tc.Namespace = "default"
			tac := newTidbClusterAutoScaler(tc)
			if testcase.haveRef {
				tc.Status.AutoScaler = &v1alpha1.TidbClusterAutoScalerRef{
					Name:      tac.Name,
					Namespace: tac.Namespace,
				}
			} else {
				tc.Status.AutoScaler = nil
			}
			if !testcase.correctRef {
				tac.Spec.Cluster.Name = "1234"
			}

			if testcase.autoScalerExisted {
				scalerInder.Add(tac)
			}
			err := tsm.syncAutoScalerRef(tc)
			g.Expect(err).ShouldNot(HaveOccurred())
			if testcase.expectedStatusRef == nil {
				g.Expect(tc.Status.AutoScaler).Should(BeNil())
			} else {
				g.Expect(tc.Status.AutoScaler).ShouldNot(BeNil())
				g.Expect(tc.Status.AutoScaler.Name).Should(Equal(testcase.expectedStatusRef.Name))
				g.Expect(tc.Status.AutoScaler.Namespace).Should(Equal(testcase.expectedStatusRef.Namespace))
			}
		})
	}
}

func newFakeTidbClusterStatusManager() (*TidbClusterStatusManager, kubernetes.Interface, *fake.Clientset, cache.Indexer) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, 0)
	scalerInformer := informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers()
	scalerInder := scalerInformer.Informer().GetIndexer()
	tikvGroupInformer := informerFactory.Pingcap().V1alpha1().TiKVGroups()
	return NewTidbClusterStatusManager(kubeCli, cli, scalerInformer.Lister(), tikvGroupInformer.Lister()), kubeCli, cli, scalerInder
}

func newTidbClusterAutoScaler(tc *v1alpha1.TidbCluster) *v1alpha1.TidbClusterAutoScaler {
	tac := &v1alpha1.TidbClusterAutoScaler{
		Spec: v1alpha1.TidbClusterAutoScalerSpec{
			Cluster: v1alpha1.TidbClusterRef{
				Namespace: tc.Namespace,
				Name:      tc.Name,
			},
		},
	}
	tac.Name = "auto-scaler"
	tac.Namespace = "default"
	return tac
}
