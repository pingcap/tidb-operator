// Copyright 2024 PingCAP, Inc.
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
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTiProxyMemberManagerSetLabels(t *testing.T) {
	g := NewGomegaWithT(t)
	type Member struct {
		name       string
		node       string
		nodeLabels map[string]string
		unHealth   bool
	}

	type testcase struct {
		name           string
		tiproxyVersion string
		members        []Member
		missingNodes   map[string]struct{}
		labels         []string
		errExpectFn    func(*GomegaWithT, error)
		setCount       int
		labelSetFailed bool
		getConfigErr   error
	}
	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		if test.tiproxyVersion == "" {
			test.tiproxyVersion = tiproxySupportLabelsMinVersion
		}
		tc.Spec.TiProxy.Version = &test.tiproxyVersion
		tc.Spec.TiProxy.BaseImage = "pingcap/tiproxy"
		pmm, indexers := newFakeTiProxyMemberManager()
		tiproxyCtl := pmm.deps.ProxyControl.(*controller.FakeTiProxyControl)
		pdControl := pmm.deps.PDControl.(*pdapi.FakePDControl)
		pdClient := controller.NewFakePDClient(pdControl, tc)
		tc.Status.TiProxy.Members = make(map[string]v1alpha1.TiProxyMember)
		for i, m := range test.members {
			name := m.name
			if len(name) == 0 {
				name = fmt.Sprintf("test-tiproxy-%d", i)
			}
			tc.Status.TiProxy.Members[name] = v1alpha1.TiProxyMember{
				Name:   name,
				Health: !m.unHealth,
			}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: metav1.NamespaceDefault,
				},
				Spec: corev1.PodSpec{
					NodeName: m.node,
				},
			}
			indexers.pod.Add(pod)

			if _, ok := test.missingNodes[m.node]; !ok {
				if len(m.nodeLabels) == 0 {
					m.nodeLabels = map[string]string{
						"region":                      "region",
						"topology.kubernetes.io/zone": "zone",
						"rack":                        "rack",
						corev1.LabelHostname:          "host",
					}
				}
				node := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name:   m.node,
						Labels: m.nodeLabels,
					},
				}
				indexers.node.Add(node)
			}
		}
		if test.getConfigErr != nil {
			pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, test.getConfigErr
			})
		} else {
			pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
				labels := test.labels
				if len(labels) == 0 {
					labels = []string{"topology.kubernetes.io/zone", corev1.LabelHostname}
				}
				return &pdapi.PDConfigFromAPI{
					Replication: &pdapi.PDReplicationConfig{
						LocationLabels: labels,
					},
				}, nil
			})
		}

		if test.labelSetFailed {
			tiproxyCtl.SetLabelsErr(fmt.Errorf("mock label set failed"))
		}

		setCount, err := pmm.setLabelsForTiProxy(tc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		} else {
			g.Expect(err).To(BeNil())
			g.Expect(setCount).To(Equal(test.setCount))
		}
	}
	tests := []testcase{
		{
			name:         "get pd config return error",
			getConfigErr: fmt.Errorf("failed to get pd config"),
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to get pd config")).To(BeTrue())
			},
		},
		{
			name:   "zone labels not found",
			labels: []string{"unknown"},
		},
		{
			name: "all members not health",
			members: []Member{
				{
					node:     "node-1",
					unHealth: true,
				},
				{
					node:     "node-2",
					unHealth: true,
				},
			},
		},
		{
			name: "illegal pod ordinal",
			members: []Member{
				{
					name: "arbitrary-name",
					node: "node-1",
				},
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "parse ordinal from pod name 'arbitrary-name' failed")).To(BeTrue())
			},
		},
		{
			name: "don't have node",
			members: []Member{
				{
					name: "test-tiproxy-0",
					node: "node-1",
				},
			},
			missingNodes: map[string]struct{}{
				"node-1": {},
			},
		},
		{
			name: "node label don't match",
			members: []Member{
				{
					name: "test-tiproxy-0",
					node: "node-1",
					nodeLabels: map[string]string{
						"test": "123",
					},
				},
			},
		},
		{
			name: "set labels failed",
			members: []Member{
				{
					node: "node-1",
				},
				{
					node: "node-2",
				},
			},
			setCount: 2,
		},
		{
			name:           "skip old version tiproxy",
			tiproxyVersion: "abcdefj", // sha tag
			members: []Member{
				{
					node: "node-1",
				},
				{
					node: "node-2",
				},
			},
		},
		{
			name:           "new version tiproxy",
			tiproxyVersion: "v1.1.0-alpha",
			members: []Member{
				{
					node: "node-1",
				},
			},
			setCount: 1,
		},
		{
			name: "skip unhealthy pods",
			members: []Member{
				{
					node: "node-1",
				},
				{
					node:     "node-2",
					unHealth: true,
				},
			},
			setCount: 1,
		},
	}

	for i := range tests {
		t.Logf(tests[i].name)
		testFn(&tests[i], t)
	}
}

func newFakeTiProxyMemberManager() (*tiproxyMemberManager, *fakeIndexers) {
	fakeDeps := controller.NewFakeDependencies()
	tmm := &tiproxyMemberManager{
		deps:      fakeDeps,
		scaler:    NewTiProxyScaler(fakeDeps),
		upgrader:  NewFakeTiProxyUpgrader(),
		suspender: suspender.NewFakeSuspender(),
	}
	indexers := &fakeIndexers{
		pod:    fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer(),
		tc:     fakeDeps.InformerFactory.Pingcap().V1alpha1().TidbClusters().Informer().GetIndexer(),
		svc:    fakeDeps.KubeInformerFactory.Core().V1().Services().Informer().GetIndexer(),
		eps:    fakeDeps.KubeInformerFactory.Core().V1().Endpoints().Informer().GetIndexer(),
		secret: fakeDeps.KubeInformerFactory.Core().V1().Secrets().Informer().GetIndexer(),
		set:    fakeDeps.KubeInformerFactory.Apps().V1().StatefulSets().Informer().GetIndexer(),
		node:   fakeDeps.KubeInformerFactory.Core().V1().Nodes().Informer().GetIndexer(),
	}
	return tmm, indexers
}
