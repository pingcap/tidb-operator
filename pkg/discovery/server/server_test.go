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

package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var (
	tc = &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{Kind: "TidbCluster", APIVersion: "v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			Namespace:       metav1.NamespaceDefault,
			ResourceVersion: "1",
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{Replicas: 3},
		},
	}
)

func TestServer(t *testing.T) {
	os.Setenv("MY_POD_NAMESPACE", "default")
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	fakePDControl := pdapi.NewFakePDControl(kubeCli)
	faleMasterControl := dmapi.NewFakeMasterControl(kubeCli)
	pdClient := pdapi.NewFakePDClient()
	s := NewServer(fakePDControl, faleMasterControl, cli, kubeCli)
	httpServer := httptest.NewServer(s.(*server).container.ServeMux)
	defer httpServer.Close()

	var lock sync.RWMutex
	pdMemberInfos := &pdapi.MembersInfo{
		Members: []*pdpb.Member{},
	}
	pdClient.AddReaction(pdapi.GetMembersActionType, func(action *pdapi.Action) (interface{}, error) {
		lock.RLock()
		defer lock.RUnlock()
		if len(pdMemberInfos.Members) <= 0 {
			return nil, fmt.Errorf("no members yet")
		}
		// as pdMemberInfos.Members maybe modified, we must return a copy
		ret := *pdMemberInfos
		return &ret, nil
	})
	cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
	fakePDControl.SetPDClient(pdapi.Namespace(tc.Namespace), tc.Name, pdClient)

	var (
		initial int32
		join    int32
	)

	errg, _ := errgroup.WithContext(context.Background())

	for i := 0; i < 3; i++ {
		i := i
		errg.Go(func() error {
			for {
				svc := fmt.Sprintf(`foo-pd-%d.foo-pd-peer.default.svc:2380`, i)
				url := httpServer.URL + fmt.Sprintf("/new/%s", base64.StdEncoding.EncodeToString([]byte(svc)))
				resp, err := http.Get(url)
				if err != nil {
					return err
				}
				if resp.StatusCode != http.StatusOK {
					time.Sleep(time.Millisecond * 100)
					continue
				}
				data, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				lock.Lock()
				pdMemberInfos.Members = append(pdMemberInfos.Members, &pdpb.Member{
					Name: svc,
					PeerUrls: []string{
						svc,
					},
				})
				lock.Unlock()
				if strings.HasPrefix(string(data), "--join=") {
					atomic.AddInt32(&join, 1)
				} else if strings.HasPrefix(string(data), "--initial-cluster=") {
					atomic.AddInt32(&initial, 1)
				}
				return nil
			}
		})
	}

	err := errg.Wait()
	if err != nil {
		t.Errorf("get pd info failed: %v", err)
	}

	if initial != 1 {
		t.Errorf("initial expects 1, got %d", initial)
	}
	if join != 2 {
		t.Errorf("join expects 2, got %d", join)
	}
}
