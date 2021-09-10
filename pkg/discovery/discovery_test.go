// Copyright 2018 PingCAP, Inc.
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

package discovery

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestDiscoveryDiscovery(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name         string
		ns           string
		url          string
		clusters     map[string]*clusterInfo
		tc           *v1alpha1.TidbCluster
		getMembersFn func() (*pdapi.MembersInfo, error)
		expectFn     func(*GomegaWithT, *tidbDiscovery, string, error)
	}
	testFn := func(test testcase, t *testing.T) {
		cli := fake.NewSimpleClientset()
		kubeCli := kubefake.NewSimpleClientset()
		informer := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
		fakePDControl := pdapi.NewFakePDControl(informer.Core().V1().Secrets().Lister())
		fakeMasterControl := dmapi.NewFakeMasterControl(informer.Core().V1().Secrets().Lister())
		pdClient := pdapi.NewFakePDClient()
		if test.tc != nil {
			cli.PingcapV1alpha1().TidbClusters(test.tc.Namespace).Create(context.TODO(), test.tc, metav1.CreateOptions{})
			fakePDControl.SetPDClient(pdapi.Namespace(test.tc.GetNamespace()), test.tc.GetName(), pdClient)
		}
		pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
			return test.getMembersFn()
		})

		td := NewTiDBDiscovery(fakePDControl, fakeMasterControl, cli, kubeCli)
		td.(*tidbDiscovery).clusters = test.clusters

		os.Setenv("MY_POD_NAMESPACE", test.ns)
		re, err := td.Discover(test.url)
		test.expectFn(g, td.(*tidbDiscovery), re, err)
	}
	tests := []testcase{
		{
			name:     "advertisePeerUrl is empty",
			ns:       "default",
			url:      "",
			clusters: map[string]*clusterInfo{},
			tc:       newTC(),
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "advertisePeerUrl is empty")).To(BeTrue())
				g.Expect(len(td.clusters)).To(BeZero())
			},
		},
		{
			name:     "advertisePeerUrl is wrong",
			ns:       "default",
			url:      "demo-pd-0.demo-pd-peer.default:2380",
			clusters: map[string]*clusterInfo{},
			tc:       newTC(),
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "advertisePeerUrl format is wrong: ")).To(BeTrue())
				g.Expect(len(td.clusters)).To(BeZero())
			},
		},
		{
			name:     "namespace is wrong",
			ns:       "default1",
			url:      "demo-pd-0.demo-pd-peer.default.svc:2380",
			clusters: map[string]*clusterInfo{},
			tc:       newTC(),
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "is not equal to discovery namespace:")).To(BeTrue())
				g.Expect(len(td.clusters)).To(BeZero())
			},
		},
		{
			name:     "failed to get tidbcluster",
			ns:       "default",
			url:      "demo-pd-0.demo-pd-peer.default.svc:2380",
			clusters: map[string]*clusterInfo{},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
				g.Expect(len(td.clusters)).To(BeZero())
			},
		},
		{
			name:     "failed to get members",
			ns:       "default",
			url:      "demo-pd-0.demo-pd-peer.default.svc:2380",
			clusters: map[string]*clusterInfo{},
			tc:       newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return nil, fmt.Errorf("get members failed")
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "get members failed")).To(BeTrue())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-0"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "resourceVersion changed",
			ns:   "default",
			url:  "demo-pd-0.demo-pd-peer.default.svc:2380",
			tc:   newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return nil, fmt.Errorf("getMembers failed")
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "2",
					peers: map[string]struct{}{
						"demo-pd-0": {},
						"demo-pd-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "getMembers failed")).To(BeTrue())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-0"]).To(Equal(struct{}{}))
			},
		},
		{
			name:     "1 cluster, first ordinal, there are no pd members",
			ns:       "default",
			url:      "demo-pd-0.demo-pd-peer.default.svc:2380",
			clusters: map[string]*clusterInfo{},
			tc:       newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return nil, fmt.Errorf("there are no pd members")
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "there are no pd members")).To(BeTrue())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-0"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "1 cluster, second ordinal, there are no pd members",
			ns:   "default",
			url:  "demo-pd-1.demo-pd-peer.default.svc:2380",
			tc:   newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return nil, fmt.Errorf("there are no pd members 2")
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-0": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "there are no pd members 2")).To(BeTrue())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(2))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-0"]).To(Equal(struct{}{}))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-1"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "1 cluster, third ordinal, return the initial-cluster args",
			ns:   "default",
			url:  "demo-pd-2.demo-pd-peer.default.svc:2380",
			tc:   newTC(),
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-0": {},
						"demo-pd-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(2))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-0"]).To(Equal(struct{}{}))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-1"]).To(Equal(struct{}{}))
				g.Expect(s).To(Equal("--initial-cluster=demo-pd-2=http://demo-pd-2.demo-pd-peer.default.svc:2380"))
			},
		},
		{
			name: "1 cluster, the first ordinal second request, get members failed",
			ns:   "default",
			url:  "demo-pd-0.demo-pd-peer.default.svc:2380",
			tc:   newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return nil, fmt.Errorf("there are no pd members 3")
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-0": {},
						"demo-pd-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "there are no pd members 3")).To(BeTrue())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(2))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-0"]).To(Equal(struct{}{}))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-1"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "1 cluster, the first ordinal third request, get members success",
			ns:   "default",
			url:  "demo-pd-0.demo-pd-peer.default.svc:2380",
			tc:   newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							PeerUrls: []string{"demo-pd-2.demo-pd-peer.default.svc:2380"},
						},
					},
				}, nil
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-0": {},
						"demo-pd-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-1"]).To(Equal(struct{}{}))
				g.Expect(s).To(Equal("--join=demo-pd-2.demo-pd-peer.default.svc:2379"))
			},
		},
		{
			name: "1 cluster, the second ordinal second request, get members success",
			ns:   "default",
			url:  "demo-pd-1.demo-pd-peer.default.svc:2380",
			tc:   newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							PeerUrls: []string{"demo-pd-0.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-2.demo-pd-peer.default.svc:2380"},
						},
					},
				}, nil
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(0))
				g.Expect(s).To(Equal("--join=demo-pd-0.demo-pd-peer.default.svc:2379,demo-pd-2.demo-pd-peer.default.svc:2379"))
			},
		},
		{
			name: "1 cluster, the fourth ordinal request, get members success",
			ns:   "default",
			url:  "demo-pd-3.demo-pd-peer.default.svc:2380",
			tc: func() *v1alpha1.TidbCluster {
				tc := newTC()
				tc.Spec.PD.Replicas = 5
				return tc
			}(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							PeerUrls: []string{"demo-pd-0.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-1.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-2.demo-pd-peer.default.svc:2380"},
						},
					},
				}, nil
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers:           map[string]struct{}{},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(0))
				g.Expect(s).To(Equal("--join=demo-pd-0.demo-pd-peer.default.svc:2379,demo-pd-1.demo-pd-peer.default.svc:2379,demo-pd-2.demo-pd-peer.default.svc:2379"))
			},
		},
		{
			name: "1 cluster, the request with clusterDomain, get members success",
			ns:   "default",
			url:  "demo-pd-0.demo-pd-peer.default.svc.cluster.local:2380",
			tc:   newTC(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							PeerUrls: []string{"demo-pd-2.demo-pd-peer.default.svc.cluster.local:2380"},
						},
					},
				}, nil
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-0": {},
						"demo-pd-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters)).To(Equal(1))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.clusters["default/demo"].peers["demo-pd-1"]).To(Equal(struct{}{}))
				g.Expect(s).To(Equal("--join=demo-pd-2.demo-pd-peer.default.svc.cluster.local:2379"))
			},
		},
		{
			name: "2 clusters, the five ordinal request, get members success",
			ns:   "default",
			url:  "demo-pd-3.demo-pd-peer.default.svc:2380",
			tc: func() *v1alpha1.TidbCluster {
				tc := newTC()
				tc.Spec.PD.Replicas = 5
				return tc
			}(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							PeerUrls: []string{"demo-pd-0.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-1.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-2.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-3.demo-pd-peer.default.svc:2380"},
						},
					},
				}, nil
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers:           map[string]struct{}{},
				},
				"default/demo-1": {
					peers: map[string]struct{}{
						"demo-1-pd-0": {},
						"demo-1-pd-1": {},
						"demo-1-pd-2": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters)).To(Equal(2))
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(0))
				g.Expect(len(td.clusters["default/demo-1"].peers)).To(Equal(3))
				g.Expect(s).To(Equal("--join=demo-pd-0.demo-pd-peer.default.svc:2379,demo-pd-1.demo-pd-peer.default.svc:2379,demo-pd-2.demo-pd-peer.default.svc:2379,demo-pd-3.demo-pd-peer.default.svc:2379"))
			},
		},
		{
			name: "pdAddresses exists, 3 pd replicas, the 3rd pd send request",
			ns:   "default",
			url:  "demo-pd-2.demo-pd-peer.default.svc:2380",
			tc: func() *v1alpha1.TidbCluster {
				tc := newTC()
				tc.Spec.PDAddresses = []string{"http://address0:2379", "http://address1:2379", "http://address2:2379"}
				return tc
			}(),
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-0": {},
						"demo-pd-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(2))
				g.Expect(s).To(Equal("--join=http://address0:2379,http://address1:2379,http://address2:2379"))
			},
		},
		{
			name: "pdAddresses exists, 3 pd replicas, get members success, the 1st pd send request",
			ns:   "default",
			url:  "demo-pd-0.demo-pd-peer.default.svc:2380",
			tc: func() *v1alpha1.TidbCluster {
				tc := newTC()
				tc.Spec.PDAddresses = []string{
					"http://address0:2379",
					"http://address1:2379",
					"http://address2:2379",
				}
				return tc
			}(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							PeerUrls: []string{"http://address0:2380"},
						},
						{
							PeerUrls: []string{"http://address1:2380"},
						},
						{
							PeerUrls: []string{"http://address2:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-1.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"demo-pd-2.demo-pd-peer.default.svc:2380"},
						},
					},
				}, nil
			},
			clusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-pd-0": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.clusters["default/demo"].peers)).To(Equal(0))
				g.Expect(s).To(Equal("--join=http://address0:2379,http://address1:2379,http://address2:2379,demo-pd-1.demo-pd-peer.default.svc:2379,demo-pd-2.demo-pd-peer.default.svc:2379"))
			},
		},
		{
			name: "skip initialize when PD on initial cluster failover for cross-region clusters",
			ns:   "default",
			url:  "demo-pd-3.demo-pd-peer.default.svc:2380",
			tc: func() *v1alpha1.TidbCluster {
				tc := newTC()
				tc.Status.PD.PeerMembers = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", ClientURL: "http://pd-0.pd.pingcap.cluster2.com:2379", Health: true},
				}
				return tc
			}(),
			getMembersFn: func() (*pdapi.MembersInfo, error) {
				return &pdapi.MembersInfo{
					Members: []*pdpb.Member{
						{
							PeerUrls: []string{"demo-pd-3.demo-pd-peer.default.svc:2380"},
						},
						{
							PeerUrls: []string{"pd-0.pd.pingcap.cluster2.com:2380"},
						},
					},
				}, nil
			},
			clusters: map[string]*clusterInfo{},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("--join=demo-pd-3.demo-pd-peer.default.svc:2379,pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestDiscoveryDMDiscovery(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name         string
		ns           string
		url          string
		dmClusters   map[string]*clusterInfo
		dc           *v1alpha1.DMCluster
		getMastersFn func() ([]*dmapi.MastersInfo, error)
		expectFn     func(*GomegaWithT, *tidbDiscovery, string, error)
	}
	testFn := func(test testcase, t *testing.T) {
		cli := fake.NewSimpleClientset()
		kubeCli := kubefake.NewSimpleClientset()
		informer := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
		fakePDControl := pdapi.NewFakePDControl(informer.Core().V1().Secrets().Lister())
		fakeMasterControl := dmapi.NewFakeMasterControl(informer.Core().V1().Secrets().Lister())
		masterClient := dmapi.NewFakeMasterClient()
		if test.dc != nil {
			cli.PingcapV1alpha1().DMClusters(test.dc.Namespace).Create(context.TODO(), test.dc, metav1.CreateOptions{})
			fakeMasterControl.SetMasterClient(test.dc.GetNamespace(), test.dc.GetName(), masterClient)
		}
		masterClient.AddReaction(dmapi.GetMastersActionType, func(action *dmapi.Action) (interface{}, error) {
			return test.getMastersFn()
		})

		td := NewTiDBDiscovery(fakePDControl, fakeMasterControl, cli, kubeCli)
		td.(*tidbDiscovery).dmClusters = test.dmClusters

		os.Setenv("MY_POD_NAMESPACE", test.ns)
		re, err := td.DiscoverDM(test.url)
		test.expectFn(g, td.(*tidbDiscovery), re, err)
	}
	tests := []testcase{
		{
			name:       "advertisePeerUrl is empty",
			ns:         "default",
			url:        "",
			dmClusters: map[string]*clusterInfo{},
			dc:         newDC(),
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "advertisePeerUrl is empty")).To(BeTrue())
				g.Expect(len(td.clusters)).To(BeZero())
			},
		},
		{
			name:       "advertisePeerUrl is wrong",
			ns:         "default",
			url:        "demo-dm-master-0.demo-dm-master-peer.svc:8291",
			dmClusters: map[string]*clusterInfo{},
			dc:         newDC(),
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "advertisePeerUrl format is wrong: ")).To(BeTrue())
				g.Expect(len(td.clusters)).To(BeZero())
			},
		},
		{
			name:       "failed to get tidbcluster",
			ns:         "default",
			url:        "demo-dm-master-0.demo-dm-master-peer:8291",
			dmClusters: map[string]*clusterInfo{},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
				g.Expect(len(td.clusters)).To(BeZero())
			},
		},
		{
			name:       "failed to get members",
			ns:         "default",
			url:        "demo-dm-master-0.demo-dm-master-peer:8291",
			dmClusters: map[string]*clusterInfo{},
			dc:         newDC(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return nil, fmt.Errorf("get members failed")
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "get members failed")).To(BeTrue())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-0"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "resourceVersion changed",
			ns:   "default",
			url:  "demo-dm-master-0.demo-dm-master-peer:8291",
			dc:   newDC(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return nil, fmt.Errorf("getMembers failed")
			},
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "2",
					peers: map[string]struct{}{
						"demo-dm-master-0": {},
						"demo-dm-master-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "getMembers failed")).To(BeTrue())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-0"]).To(Equal(struct{}{}))
			},
		},
		{
			name:       "1 cluster, first ordinal, there are no dm-master members",
			ns:         "default",
			url:        "demo-dm-master-0.demo-dm-master-peer:8291",
			dmClusters: map[string]*clusterInfo{},
			dc:         newDC(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return nil, fmt.Errorf("there are no dm-master members")
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "there are no dm-master members")).To(BeTrue())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-0"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "1 cluster, second ordinal, there are no dm-master members",
			ns:   "default",
			url:  "demo-dm-master-1.demo-dm-master-peer:8291",
			dc:   newDC(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return nil, fmt.Errorf("there are no dm-master members 2")
			},
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-dm-master-0": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "there are no dm-master members 2")).To(BeTrue())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(2))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-0"]).To(Equal(struct{}{}))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-1"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "1 cluster, third ordinal, return the initial-cluster args",
			ns:   "default",
			url:  "demo-dm-master-2.demo-dm-master-peer:8291",
			dc:   newDC(),
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-dm-master-0": {},
						"demo-dm-master-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(2))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-0"]).To(Equal(struct{}{}))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-1"]).To(Equal(struct{}{}))
				g.Expect(s).To(Equal("--initial-cluster=demo-dm-master-2=http://demo-dm-master-2.demo-dm-master-peer:8291"))
			},
		},
		{
			name: "1 cluster, the first ordinal second request, get members failed",
			ns:   "default",
			url:  "demo-dm-master-0.demo-dm-master-peer:8291",
			dc:   newDC(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return nil, fmt.Errorf("there are no dm-master members 3")
			},
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-dm-master-0": {},
						"demo-dm-master-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "there are no dm-master members 3")).To(BeTrue())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(2))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-0"]).To(Equal(struct{}{}))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-1"]).To(Equal(struct{}{}))
			},
		},
		{
			name: "1 cluster, the first ordinal third request, get members success",
			ns:   "default",
			url:  "demo-dm-master-0.demo-dm-master-peer:8291",
			dc:   newDC(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return []*dmapi.MastersInfo{
					{
						PeerURLs: []string{"demo-dm-master-2.demo-dm-master-peer:8291"},
					},
				}, nil
			},
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-dm-master-0": {},
						"demo-dm-master-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(1))
				g.Expect(td.dmClusters["default/demo"].peers["demo-dm-master-1"]).To(Equal(struct{}{}))
				g.Expect(s).To(Equal("--join=demo-dm-master-2.demo-dm-master-peer:8261"))
			},
		},
		{
			name: "1 cluster, the second ordinal second request, get members success",
			ns:   "default",
			url:  "demo-dm-master-1.demo-dm-master-peer:8291",
			dc:   newDC(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return []*dmapi.MastersInfo{
					{
						PeerURLs: []string{"demo-dm-master-0.demo-dm-master-peer:8291"},
					},
					{
						PeerURLs: []string{"demo-dm-master-2.demo-dm-master-peer:8291"},
					},
				}, nil
			},
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers: map[string]struct{}{
						"demo-dm-master-1": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(0))
				g.Expect(s).To(Equal("--join=demo-dm-master-0.demo-dm-master-peer:8261,demo-dm-master-2.demo-dm-master-peer:8261"))
			},
		},
		{
			name: "1 cluster, the fourth ordinal request, get members success",
			ns:   "default",
			url:  "demo-dm-master-3.demo-dm-master-peer:8291",
			dc: func() *v1alpha1.DMCluster {
				dc := newDC()
				dc.Spec.Master.Replicas = 5
				return dc
			}(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return []*dmapi.MastersInfo{
					{
						PeerURLs: []string{"demo-dm-master-0.demo-dm-master-peer:8291"},
					},
					{
						PeerURLs: []string{"demo-dm-master-1.demo-dm-master-peer:8291"},
					},
					{
						PeerURLs: []string{"demo-dm-master-2.demo-dm-master-peer:8291"},
					},
				}, nil
			},
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers:           map[string]struct{}{},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.dmClusters)).To(Equal(1))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(0))
				g.Expect(s).To(Equal("--join=demo-dm-master-0.demo-dm-master-peer:8261,demo-dm-master-1.demo-dm-master-peer:8261,demo-dm-master-2.demo-dm-master-peer:8261"))
			},
		},
		{
			name: "2 clusters, the five ordinal request, get members success",
			ns:   "default",
			url:  "demo-dm-master-3.demo-dm-master-peer:8291",
			dc: func() *v1alpha1.DMCluster {
				dc := newDC()
				dc.Spec.Master.Replicas = 5
				return dc
			}(),
			getMastersFn: func() ([]*dmapi.MastersInfo, error) {
				return []*dmapi.MastersInfo{
					{
						PeerURLs: []string{"demo-dm-master-0.demo-dm-master-peer:8291"},
					},
					{
						PeerURLs: []string{"demo-dm-master-1.demo-dm-master-peer:8291"},
					},
					{
						PeerURLs: []string{"demo-dm-master-2.demo-dm-master-peer:8291"},
					},
					{
						PeerURLs: []string{"demo-dm-master-3.demo-dm-master-peer:8291"},
					},
				}, nil
			},
			dmClusters: map[string]*clusterInfo{
				"default/demo": {
					resourceVersion: "1",
					peers:           map[string]struct{}{},
				},
				"default/demo-1": {
					peers: map[string]struct{}{
						"demo-1-dm-master-0": {},
						"demo-1-dm-master-1": {},
						"demo-1-dm-master-2": {},
					},
				},
			},
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(td.dmClusters)).To(Equal(2))
				g.Expect(len(td.dmClusters["default/demo"].peers)).To(Equal(0))
				g.Expect(len(td.dmClusters["default/demo-1"].peers)).To(Equal(3))
				g.Expect(s).To(Equal("--join=demo-dm-master-0.demo-dm-master-peer:8261,demo-dm-master-1.demo-dm-master-peer:8261,demo-dm-master-2.demo-dm-master-peer:8261,demo-dm-master-3.demo-dm-master-peer:8261"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestDiscoveryVerifyPDEndpoint(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name          string
		ns            string
		url           string
		tls           bool
		inclusterPD   bool
		peerclusterPD bool
		expectFn      func(*GomegaWithT, *tidbDiscovery, string, error)
	}
	testFn := func(test testcase, t *testing.T) {
		cli := fake.NewSimpleClientset()
		kubeCli := kubefake.NewSimpleClientset()
		informer := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
		fakePDControl := pdapi.NewFakePDControl(informer.Core().V1().Secrets().Lister())
		fakeMasterControl := dmapi.NewFakeMasterControl(informer.Core().V1().Secrets().Lister())
		tc := newTC()

		ns := "default"
		os.Setenv("MY_POD_NAMESPACE", "default")

		if test.tls {
			pdClientCluster1 := controller.NewFakePDClientWithAddress(fakePDControl, "demo-pd")
			pdClientCluster2 := controller.NewFakePDClientWithAddress(fakePDControl, "pd-0.pd.pingcap.cluster2.com")

			tc.Status.PD.PeerMembers = map[string]v1alpha1.PDMember{
				"pd-0.pd.pingcap.cluster2.com": {Name: "pd-0.pd.pingcap.cluster2.com", ClientURL: "https://pd-0.pd.pingcap.cluster2.com:2379", Health: true},
			}
			if test.inclusterPD {
				pdClientCluster1.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
						{Name: "pd-0", MemberID: uint64(1), ClientUrls: []string{"https://pd-0.pd.pingcap.cluster1.com:2379"}, Health: true},
					}}, nil
				})
			} else {
				pdClientCluster1.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, fmt.Errorf("Fake cluster 1 PD crashed")
				})
			}

			if test.peerclusterPD {
				pdClientCluster2.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
						{Name: "pd-0", MemberID: uint64(1), ClientUrls: []string{"https://pd-0.pd.pingcap.cluster2.com:2379"}, Health: true},
					}}, nil
				})
			} else {
				tc.Status.PD.PeerMembers = map[string]v1alpha1.PDMember{
					"pd-0.pd.pingcap.cluster2.com": {Name: "pd-0.pd.pingcap.cluster2.com", ClientURL: "https://pd-0.pd.pingcap.cluster2.com:2379", Health: false},
				}
				pdClientCluster2.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, fmt.Errorf("Fake cluster 2 PD crashed")
				})
			}
		} else {
			pdClientCluster1 := controller.NewFakePDClientWithAddress(fakePDControl, "demo-pd")
			pdClientCluster2 := controller.NewFakePDClientWithAddress(fakePDControl, "pd-0.pd.pingcap.cluster2.com")

			tc.Status.PD.PeerMembers = map[string]v1alpha1.PDMember{
				"pd-0.pd.pingcap.cluster2.com": {Name: "pd-0.pd.pingcap.cluster2.com", ClientURL: "http://pd-0.pd.pingcap.cluster2.com:2379", Health: true},
			}
			if test.inclusterPD {
				pdClientCluster1.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
						{Name: "pd-0", MemberID: uint64(1), ClientUrls: []string{"http://pd-0.pd.pingcap.cluster1.com:2379"}, Health: true},
					}}, nil
				})
			} else {
				pdClientCluster1.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, fmt.Errorf("Fake cluster 1 PD crashed")
				})
			}

			if test.peerclusterPD {
				pdClientCluster2.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{
						{Name: "pd-0", MemberID: uint64(1), ClientUrls: []string{"http://pd-0.pd.pingcap.cluster2.com:2379"}, Health: true},
					}}, nil
				})
			} else {
				tc.Status.PD.PeerMembers = map[string]v1alpha1.PDMember{
					"pd-0.pd.pingcap.cluster2.com": {Name: "pd-0.pd.pingcap.cluster2.com", ClientURL: "http://pd-0.pd.pingcap.cluster2.com:2379", Health: false},
				}
				pdClientCluster2.AddReaction(pdapi.GetHealthActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, fmt.Errorf("Fake cluster 2 PD crashed")
				})
			}
		}

		cli.PingcapV1alpha1().TidbClusters(ns).Create(context.TODO(), tc, metav1.CreateOptions{})
		td := NewTiDBDiscovery(fakePDControl, fakeMasterControl, cli, kubeCli)

		os.Setenv("MY_POD_NAMESPACE", test.ns)
		re, err := td.VerifyPDEndpoint(test.url)
		test.expectFn(g, td.(*tidbDiscovery), re, err)
	}
	tests := []testcase{
		{
			name:          "tidb requests, tls off, in-cluster PD is enabled, and peer-cluster PD is enabled",
			ns:            "default",
			url:           "demo-pd:2379",
			tls:           false,
			inclusterPD:   true,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("demo-pd:2379,pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tidb requests, tls off, in-cluster PD disabled, and peer-cluster PD enabled",
			ns:            "default",
			url:           "demo-pd:2379",
			tls:           false,
			inclusterPD:   false,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("demo-pd:2379,pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tidb requests, tls off, in-cluster PD disabled, and peer-cluster PD disabled",
			ns:            "default",
			url:           "demo-pd:2379",
			tls:           false,
			inclusterPD:   false,
			peerclusterPD: false,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("demo-pd:2379"))
			},
		},
		{
			name:          "tidb requests, tls on, in-cluster PD is enabled, and peer-cluster PD is enabled",
			ns:            "default",
			url:           "demo-pd:2379",
			tls:           true,
			inclusterPD:   true,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("demo-pd:2379,pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tidb requests, tls on, in-cluster PD disabled, and peer-cluster PD enabled",
			ns:            "default",
			url:           "demo-pd:2379",
			tls:           true,
			inclusterPD:   false,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("demo-pd:2379,pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tidb requests, tls on, in-cluster PD disabled, and peer-cluster PD disabled",
			ns:            "default",
			url:           "demo-pd:2379",
			tls:           true,
			inclusterPD:   false,
			peerclusterPD: false,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("demo-pd:2379"))
			},
		},
		{
			name:          "tikv requests, tls off, in-cluster PD is enabled, and peer-cluster PD is enabled",
			ns:            "default",
			url:           "http://demo-pd:2379",
			tls:           false,
			inclusterPD:   true,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("http://demo-pd:2379,http://pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tikv requests, tls off, in-cluster PD disabled, and peer-cluster PD enabled",
			ns:            "default",
			url:           "http://demo-pd:2379",
			tls:           false,
			inclusterPD:   false,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("http://demo-pd:2379,http://pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tikv requests, tls off, in-cluster PD disabled, and peer-cluster PD disabled",
			ns:            "default",
			url:           "http://demo-pd:2379",
			tls:           false,
			inclusterPD:   false,
			peerclusterPD: false,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("http://demo-pd:2379"))
			},
		},
		{
			name:          "tikv requests, tls on, in-cluster PD is enabled, and peer-cluster PD is enabled",
			ns:            "default",
			url:           "https://demo-pd:2379",
			tls:           true,
			inclusterPD:   true,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("https://demo-pd:2379,https://pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tikv requests, tls on, in-cluster PD disabled, and peer-cluster PD enabled",
			ns:            "default",
			url:           "https://demo-pd:2379",
			tls:           true,
			inclusterPD:   false,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("https://demo-pd:2379,https://pd-0.pd.pingcap.cluster2.com:2379"))
			},
		},
		{
			name:          "tikv requests, tls on, in-cluster PD disabled, and peer-cluster PD disabled",
			ns:            "default",
			url:           "https://demo-pd:2379",
			tls:           true,
			inclusterPD:   false,
			peerclusterPD: false,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(s).To(Equal("https://demo-pd:2379"))
			},
		},
		{
			name:          "tidbcluster name not exist",
			ns:            "default",
			url:           "non-exists-demo-pd:2379",
			tls:           false,
			inclusterPD:   true,
			peerclusterPD: true,
			expectFn: func(g *GomegaWithT, td *tidbDiscovery, s string, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(s).To(Equal("non-exists-demo-pd:2379"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func newTC() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{Kind: "TidbCluster", APIVersion: "v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "demo",
			Namespace:       metav1.NamespaceDefault,
			ResourceVersion: "1",
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{Replicas: 3},
		},
	}
}

func newDC() *v1alpha1.DMCluster {
	return &v1alpha1.DMCluster{
		TypeMeta: metav1.TypeMeta{Kind: "DmCluster", APIVersion: "v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "demo",
			Namespace:       metav1.NamespaceDefault,
			ResourceVersion: "1",
		},
		Spec: v1alpha1.DMClusterSpec{
			Master: v1alpha1.MasterSpec{Replicas: 3},
		},
	}
}
