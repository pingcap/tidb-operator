// Copyright 2019 PingCAP, Inc.
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

package controller

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestGetPDClient(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbCluster()
	kubeCli := kubefake.NewSimpleClientset()
	informer := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	pdControl := pdapi.NewFakePDControl(informer.Core().V1().Secrets().Lister())

	type testcase struct {
		name     string
		update   func(*v1alpha1.TidbCluster)
		expectFn func(*GomegaWithT, bool)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		test.update(tc)
		test.expectFn(g, tc.PDIsAvailable())
	}
	tests := []testcase{
		{
			name: "Test GetPDClient",
			update: func(tc *v1alpha1.TidbCluster) {
			},
			expectFn: func(g *GomegaWithT, b bool) {
				pdClientCluster1 := NewFakePDClient(pdControl, tc)

				pdClientCluster1.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
						{Name: "pd-0", MemberID: uint64(1), ClientUrls: []string{"http://pd-0.pd.pingcap.cluster1.com:2379"}, Health: true},
					}}, nil
				})
				pdClient := GetPDClient(pdControl, tc)
				_, err := pdClient.GetHealth()
				g.Expect(err).To(BeNil())
			},
		},
		{
			name: "Test GetPDClient when in-cluster PD endpoint failed and peer-cluster PD endpoint works",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.PeerMembers = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", ClientURL: "http://pd-0.pd.pingcap.cluster2.com:2379", Health: true},
				}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				pdClientCluster1 := NewFakePDClient(pdControl, tc)

				pdClientCluster1.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, fmt.Errorf("Fake cluster 1 PD crashed")
				})
				pdClientCluster2 := NewFakePDClientWithAddress(pdControl, "pd-0")
				pdClientCluster2.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
						{Name: "pd-0", MemberID: uint64(1), ClientUrls: []string{"http://pd-0.pd.pingcap.cluster2.com:2379"}, Health: true},
					}}, nil
				})
				pdClient := GetPDClient(pdControl, tc)
				_, err := pdClient.GetHealth()
				g.Expect(err).To(BeNil())
			},
		},
		{
			name: "Test GetPDClient when in-cluster PD endpoint failed and peer-cluster PD endpoint failed too",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.PD.PeerMembers = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", ClientURL: "http://pd-0.pd.pingcap.cluster2.com:2379", Health: true},
				}
			},
			expectFn: func(g *GomegaWithT, b bool) {
				pdClientCluster1 := NewFakePDClient(pdControl, tc)

				pdClientCluster1.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, fmt.Errorf("Fake cluster 1 PD crashed")
				})
				pdClientCluster2 := NewFakePDClientWithAddress(pdControl, "pd-0")
				pdClientCluster2.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, fmt.Errorf("Fake cluster 2 PD crashed")
				})
				pdClient := GetPDClient(pdControl, tc)
				_, err := pdClient.GetHealth()
				g.Expect(err).To(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}
