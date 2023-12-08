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

package pdapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	pd "github.com/tikv/pd/client/http"
)

const (
	ContentTypeJSON string = "application/json"
)

func getClientServer(h func(http.ResponseWriter, *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(h))
}

func TestHealth(t *testing.T) {
	g := NewGomegaWithT(t)
	healths := []pd.MemberHealth{
		{Name: "pd1", MemberID: 1, Health: false},
		{Name: "pd2", MemberID: 2, Health: true},
		{Name: "pd3", MemberID: 3, Health: true},
	}
	healthsBytes, err := json.Marshal(healths)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		path     string
		resp     []byte
		want     []pd.MemberHealth
	}{{
		caseName: "GetHealth",
		resp:     healthsBytes,
		want:     healths,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		result, err := pdClient.GetHealth(context.TODO())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(*result).To(Equal(pd.HealthInfo{Healths: healths}))
	}
}

func TestGetConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	config := &pd.ServerConfig{
		Schedule: pd.ScheduleConfig{
			MaxStoreDownTime: pd.NewDuration(10 * time.Second),
		},
	}
	configBytes, err := json.Marshal(config)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		resp     []byte
		want     *pd.ServerConfig
	}{{
		caseName: "GetConfig",
		resp:     configBytes,
		want:     config,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		result, err := pdClient.GetConfig(context.Background())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(config))
	}

}

func TestGetCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	cluster := &metapb.Cluster{Id: 1, MaxPeerCount: 100}
	clusterBytes, err := json.Marshal(cluster)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		resp     []byte
		want     *metapb.Cluster
	}{{
		caseName: "GetCluster",
		resp:     clusterBytes,
		want:     cluster,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		result, err := pdClient.GetCluster(context.TODO())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(cluster))
	}

}

func TestGetMembers(t *testing.T) {
	g := NewGomegaWithT(t)

	member1 := &pdpb.Member{Name: "testMember1", MemberId: uint64(1)}
	member2 := &pdpb.Member{Name: "testMember2", MemberId: uint64(2)}
	members := &pd.MembersInfo{
		Members: []*pdpb.Member{
			member1,
			member2,
		},
		Leader:     member1,
		EtcdLeader: member1,
	}
	membersBytes, err := json.Marshal(members)
	if err != nil {
		t.Error(err)
	}

	tcs := []struct {
		caseName string
		resp     []byte
		want     *pd.MembersInfo
	}{
		{
			caseName: "GetMembers",
			resp:     membersBytes,
			want:     members,
		},
	}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		result, err := pdClient.GetMembers(context.TODO())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(members))
	}
}

func TestGetStores(t *testing.T) {
	g := NewGomegaWithT(t)
	store1 := pd.StoreInfo{
		Store: pd.MetaStore{ID: 1, State: int64(metapb.StoreState_Up)},
	}
	store2 := pd.StoreInfo{
		Store: pd.MetaStore{ID: 2, State: int64(metapb.StoreState_Up)},
	}
	stores := &pd.StoresInfo{
		Count: 2,
		Stores: []pd.StoreInfo{
			store1,
			store2,
		},
	}

	storesBytes, err := json.Marshal(stores)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		resp     []byte
		want     *pd.StoresInfo
	}{{
		caseName: "GetStores",
		resp:     storesBytes,
		want:     stores,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		result, err := pdClient.GetStores(context.TODO())
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(stores))
	}
}

func TestGetStore(t *testing.T) {
	g := NewGomegaWithT(t)

	id := int64(1)
	store := &pd.StoreInfo{
		Store:  pd.MetaStore{ID: id, State: int64(metapb.StoreState_Up)},
		Status: pd.StoreStatus{},
	}

	storeBytes, err := json.Marshal(store)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		id       int64
		resp     []byte
		want     *pd.StoreInfo
	}{{
		caseName: "GetStore",
		id:       id,
		resp:     storeBytes,
		want:     store,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		result, err := pdClient.GetStore(context.TODO(), uint64(tc.id))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(store))
	}
}

func TestSetStoreLabels(t *testing.T) {
	g := NewGomegaWithT(t)
	id := uint64(1)
	labels := map[string]string{"testkey": "testvalue"}
	tcs := []struct {
		caseName string
		want     bool
	}{{
		caseName: "success_SetStoreLabels",
		want:     true,
	}, {
		caseName: "failed_SetStoreLabels",
		want:     false,
	},
	}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			labels := &map[string]string{}
			err := readJSON(request.Body, labels)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(labels).To(Equal(labels), "check labels")

			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		result, _ := pdClient.SetStoreLabels(context.TODO(), id, labels)
		g.Expect(result).To(Equal(tc.want))
	}
}

func TestDeleteMember(t *testing.T) {
	g := NewGomegaWithT(t)

	tcs := []struct {
		caseName string
		exist    bool
		want     bool
	}{{
		caseName: "success_DeleteMember",
		exist:    true,
		want:     true,
	}, {
		caseName: "failed_DeleteMember",
		exist:    true,
		want:     false,
	}, {
		caseName: "delete_not_exist_member",
		exist:    false,
		want:     true,
	},
	}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			if !tc.exist {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			g.Expect(tc.exist).To(BeTrue())
			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		err := pdClient.DeleteMember(context.TODO(), "testMember")
		if tc.want && tc.exist {
			g.Expect(err).NotTo(HaveOccurred(), "check result")
		} else {
			g.Expect(err).To(HaveOccurred(), "check result")
		}
	}
}

func TestDeleteMemberByID(t *testing.T) {
	g := NewGomegaWithT(t)

	tcs := []struct {
		caseName string
		preResp  []byte
		exist    bool
		want     bool
	}{{
		caseName: "success_DeleteMemberByID",
		exist:    true,
		want:     true,
	}, {
		caseName: "failed_DeleteMemberByID",
		exist:    true,
		want:     false,
	}, {
		caseName: "delete_not_exit_member",
		exist:    false,
		want:     true,
	},
	}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			if !tc.exist {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		err := pdClient.DeleteMemberByID(context.TODO(), 1)
		if tc.want && tc.exist {
			g.Expect(err).NotTo(HaveOccurred(), "check result")
		} else {
			g.Expect(err).To(HaveOccurred(), "check result")
		}
	}
}

func TestDeleteStore(t *testing.T) {
	g := NewGomegaWithT(t)

	tcs := []struct {
		caseName string
		preResp  []byte
		exist    bool
		want     bool
	}{{
		caseName: "success_DeleteStore",
		exist:    true,
		want:     true,
	}, {
		caseName: "failed_DeleteStore",
		exist:    true,
		want:     false,
	}, {
		caseName: "delete_not_exist_store",
		exist:    true,
		want:     true,
	},
	}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			if !tc.exist {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		err := pdClient.DeleteStore(context.TODO(), 1)
		if tc.want && tc.exist {
			g.Expect(err).NotTo(HaveOccurred(), "check result")
		} else {
			g.Expect(err).To(HaveOccurred(), "check result")
		}
	}
}

func TestGetEvictLeaderSchedulersForStores(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		storeIDs []uint64
		resp     []byte
		expect   func(schedulers map[uint64]string, err error)
	}

	cases := []testcase{
		{
			name:     "all stores have evict scheduler",
			storeIDs: []uint64{1, 7, 8},
			resp: []byte(`
[
	"evict-leader-scheduler-1",
	"evict-leader-scheduler-7",
	"evict-leader-scheduler-8",
	"evict-leader-scheduler-9"
]
			`),
			expect: func(schedulers map[uint64]string, err error) {
				g.Expect(err).To(Succeed())
				storeIDs := []uint64{1, 7, 8}
				g.Expect(schedulers).To(HaveLen(len(storeIDs)))
				for storeID, scheduler := range schedulers {
					g.Expect(storeIDs).To(ContainElement(storeID), "store id isn't in storeIDs")
					g.Expect(scheduler).To(Equal(getLeaderEvictSchedulerStr(storeID)), "scheduler does not match with the store id")
				}
			},
		},
		{
			name:     "some stores have evict scheduler",
			storeIDs: []uint64{1, 7, 10},
			resp: []byte(`
[
	"evict-leader-scheduler-1",
	"evict-leader-scheduler-7",
	"evict-leader-scheduler-8",
	"evict-leader-scheduler-9"
]
			`),
			expect: func(schedulers map[uint64]string, err error) {
				g.Expect(err).To(Succeed())
				storeIDs := []uint64{1, 7, 10}
				g.Expect(len(schedulers)).To(Equal(2))
				for storeID, scheduler := range schedulers {
					g.Expect(storeIDs).To(ContainElement(storeID), "store id isn't in storeIDs")
					g.Expect(scheduler).To(Equal(getLeaderEvictSchedulerStr(storeID)), "scheduler is't corresponding to store id")
				}
			},
		},
		{
			name:     "no store have evict scheduler",
			storeIDs: []uint64{1, 7, 10},
			resp: []byte(`
[
	"evict-leader-scheduler-2",
	"evict-leader-scheduler-8",
	"evict-leader-scheduler-9"
]
			`),
			expect: func(schedulers map[uint64]string, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(schedulers).To(HaveLen(0))
			},
		},
	}

	for _, tc := range cases {
		t.Logf("test case: %s", tc.name)

		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Content-Type", ContentTypeJSON)
			w.WriteHeader(http.StatusOK)
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := pd.NewClient([]string{svc.URL})
		schedulers, err := GetEvictLeaderSchedulersForStores(context.TODO(), pdClient, tc.storeIDs...)
		tc.expect(schedulers, err)
	}
}

func readJSON(r io.ReadCloser, data interface{}) error {
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, data)
	if err != nil {
		return err
	}

	return nil
}

func checkNoError(t *testing.T, results []reflect.Value) {
	lastVal := results[len(results)-1].Interface()
	v, ok := lastVal.(error)
	if !ok {
		return
	}
	if v != nil {
		t.Errorf("expects no error, got %v", v)
	}
}

// TestGeneric is a generic test to test methods of PD Client.
func TestGeneric(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		args        []reflect.Value
		resp        []byte
		statusCode  int
		wantQuery   string
		checkResult func(t *testing.T, results []reflect.Value)
	}{
		{
			name:   "UpdateReplicationConfig",
			method: "UpdateReplicationConfig",
			args: []reflect.Value{
				reflect.ValueOf(pd.ReplicationConfig{}),
			},
			resp:        []byte(``),
			statusCode:  http.StatusOK,
			checkResult: checkNoError,
		},
		// TODO test the fix https://github.com/pingcap/tidb-operator/pull/2809
		// {
		// name:        "GetEvictLeaderSchedulers for the new PD versions",
		// method:      "GetEvictLeaderSchedulers",
		// },
		{
			name:   "GetPDLeader",
			method: "GetPDLeader",
			resp: []byte(`
{
	"name": "pd-leader",
	"member_id": 1
}
`),
			statusCode:  http.StatusOK,
			checkResult: checkNoError,
		},
		{
			name:   "TransferPDLeader",
			method: "TransferPDLeader",
			args: []reflect.Value{
				reflect.ValueOf("foo"),
			},
			resp: []byte(`
`),

			statusCode:  http.StatusOK,
			checkResult: checkNoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			server := getClientServer(func(w http.ResponseWriter, request *http.Request) {
				g.Expect(request.URL.RawQuery).To(Equal(tt.wantQuery), "check query")

				w.Header().Set("Content-Type", ContentTypeJSON)
				w.WriteHeader(tt.statusCode)
				w.Write(tt.resp)
			})
			defer server.Close()

			pdClient := pd.NewClient([]string{server.URL})
			args := []reflect.Value{
				reflect.ValueOf(pdClient),
				reflect.ValueOf(context.TODO()),
			}
			args = append(args, tt.args...)
			method, ok := reflect.TypeOf(pdClient).MethodByName(tt.method)
			if !ok {
				t.Fatalf("method %q not found", tt.method)
			}
			results := method.Func.Call(args)
			tt.checkResult(t, results)
		})
	}
}
