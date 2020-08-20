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
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		fakePDControl := pdapi.NewFakePDControl(kubeCli)
		pdClient := pdapi.NewFakePDClient()
		if test.tc != nil {
			cli.PingcapV1alpha1().TidbClusters(test.tc.Namespace).Create(test.tc)
			fakePDControl.SetPDClient(pdapi.Namespace(test.tc.GetNamespace()), test.tc.GetName(), test.tc.Spec.ClusterDomain, pdClient)
		}
		pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
			return test.getMembersFn()
		})

		td := NewTiDBDiscovery(fakePDControl, cli, kubeCli)
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
