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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
)

const (
	ContentTypeJSON string = "application/json"
)

func getClientServer(h func(http.ResponseWriter, *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(h))
}

func TestHealth(t *testing.T) {
	g := NewGomegaWithT(t)
	healths := []MemberHealth{
		{Name: "pd1", MemberID: 1, Health: false},
		{Name: "pd2", MemberID: 2, Health: true},
		{Name: "pd3", MemberID: 3, Health: true},
	}
	healthsBytes, err := json.Marshal(healths)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		path     string
		method   string
		resp     []byte
		want     []MemberHealth
	}{{
		caseName: "GetHealth",
		path:     fmt.Sprintf("/%s", healthPrefix),
		method:   "GET",
		resp:     healthsBytes,
		want:     healths,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		result, err := pdClient.GetHealth()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(&HealthInfo{healths}))
	}
}

func TestGetConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	config := &PDConfigFromAPI{
		Schedule: &PDScheduleConfig{
			MaxStoreDownTime: "10s",
		},
	}
	configBytes, err := json.Marshal(config)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		path     string
		method   string
		resp     []byte
		want     *PDConfigFromAPI
	}{{
		caseName: "GetConfig",
		path:     fmt.Sprintf("/%s", configPrefix),
		method:   "GET",
		resp:     configBytes,
		want:     config,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		result, err := pdClient.GetConfig()
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
		path     string
		method   string
		resp     []byte
		want     *metapb.Cluster
	}{{
		caseName: "GetCluster",
		path:     fmt.Sprintf("/%s", clusterIDPrefix),
		method:   "GET",
		resp:     clusterBytes,
		want:     cluster,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		result, err := pdClient.GetCluster()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(cluster))
	}

}

func TestGetMembers(t *testing.T) {
	g := NewGomegaWithT(t)

	member1 := &pdpb.Member{Name: "testMember1", MemberId: uint64(1)}
	member2 := &pdpb.Member{Name: "testMember2", MemberId: uint64(2)}
	members := &MembersInfo{
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
		path     string
		method   string
		resp     []byte
		want     *MembersInfo
	}{
		{
			caseName: "GetMembers",
			path:     fmt.Sprintf("/%s", membersPrefix),
			method:   "GET",
			resp:     membersBytes,
			want:     members,
		},
	}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		result, err := pdClient.GetMembers()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(members))
	}
}

func TestGetStores(t *testing.T) {
	g := NewGomegaWithT(t)
	store1 := &StoreInfo{
		Store:  &MetaStore{Store: &metapb.Store{Id: uint64(1), State: metapb.StoreState_Up}},
		Status: &StoreStatus{},
	}
	store2 := &StoreInfo{
		Store:  &MetaStore{Store: &metapb.Store{Id: uint64(2), State: metapb.StoreState_Up}},
		Status: &StoreStatus{},
	}
	stores := &StoresInfo{
		Count: 2,
		Stores: []*StoreInfo{
			store1,
			store2,
		},
	}

	storesBytes, err := json.Marshal(stores)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		path     string
		method   string
		resp     []byte
		want     *StoresInfo
	}{{
		caseName: "GetStores",
		path:     fmt.Sprintf("/%s", storesPrefix),
		method:   "GET",
		resp:     storesBytes,
		want:     stores,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		result, err := pdClient.GetStores()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(stores))
	}
}

func TestGetStore(t *testing.T) {
	g := NewGomegaWithT(t)

	id := uint64(1)
	store := &StoreInfo{
		Store:  &MetaStore{Store: &metapb.Store{Id: id, State: metapb.StoreState_Up}},
		Status: &StoreStatus{},
	}

	storeBytes, err := json.Marshal(store)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		path     string
		method   string
		id       uint64
		resp     []byte
		want     *StoreInfo
	}{{
		caseName: "GetStore",
		path:     fmt.Sprintf("/%s/%d", storePrefix, id),
		method:   "GET",
		id:       id,
		resp:     storeBytes,
		want:     store,
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "test method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "test url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		result, err := pdClient.GetStore(tc.id)
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
		path     string
		method   string
		want     bool
	}{{
		caseName: "success_SetStoreLabels",
		path:     fmt.Sprintf("/%s/%d/label", storePrefix, id),
		method:   "POST",
		want:     true,
	}, {
		caseName: "failed_SetStoreLabels",
		path:     fmt.Sprintf("/%s/%d/label", storePrefix, id),
		method:   "POST",
		want:     false,
	},
	}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")

			labels := &map[string]string{}
			err := readJSON(request.Body, labels)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(labels).To(Equal(labels), "check labels")

			w.Header().Set("Content-Type", ContentTypeJSON)
			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		result, _ := pdClient.SetStoreLabels(id, labels)
		g.Expect(result).To(Equal(tc.want))
	}
}

func TestDeleteMember(t *testing.T) {
	g := NewGomegaWithT(t)
	name := "testMember"
	member := &pdpb.Member{Name: name, MemberId: uint64(1)}
	membersExist := &MembersInfo{
		Members: []*pdpb.Member{
			member,
		},
		Leader:     member,
		EtcdLeader: member,
	}
	membersExistBytes, err := json.Marshal(membersExist)
	g.Expect(err).NotTo(HaveOccurred())

	membersNotExist := &MembersInfo{
		Members: []*pdpb.Member{},
	}
	membersNotExistBytes, err := json.Marshal(membersNotExist)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName  string
		prePath   string
		preMethod string
		preResp   []byte
		exist     bool
		path      string
		method    string
		want      bool
	}{{
		caseName:  "success_DeleteMember",
		prePath:   fmt.Sprintf("/%s", membersPrefix),
		preMethod: "GET",
		preResp:   membersExistBytes,
		exist:     true,
		path:      fmt.Sprintf("/%s/name/%s", membersPrefix, name),
		method:    "DELETE",
		want:      true,
	}, {
		caseName:  "failed_DeleteMember",
		prePath:   fmt.Sprintf("/%s", membersPrefix),
		preMethod: "GET",
		preResp:   membersExistBytes,
		exist:     true,
		path:      fmt.Sprintf("/%s/name/%s", membersPrefix, name),
		method:    "DELETE",
		want:      false,
	}, {
		caseName:  "delete_not_exist_member",
		prePath:   fmt.Sprintf("/%s", membersPrefix),
		preMethod: "GET",
		preResp:   membersNotExistBytes,
		exist:     false,
		path:      fmt.Sprintf("/%s/name/%s", membersPrefix, name),
		method:    "DELETE",
		want:      true,
	},
	}

	for _, tc := range tcs {
		count := 1
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			if count == 1 {
				g.Expect(request.Method).To(Equal(tc.preMethod), "check method")
				g.Expect(request.URL.Path).To(Equal(tc.prePath), "check url")
				w.Header().Set("Content-Type", ContentTypeJSON)
				w.WriteHeader(http.StatusOK)
				w.Write(tc.preResp)
				count++
				return
			}

			g.Expect(tc.exist).To(BeTrue())
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")
			w.Header().Set("Content-Type", ContentTypeJSON)
			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		err := pdClient.DeleteMember(name)
		if tc.want {
			g.Expect(err).NotTo(HaveOccurred(), "check result")
		} else {
			g.Expect(err).To(HaveOccurred(), "check result")
		}
	}
}

func TestDeleteMemberByID(t *testing.T) {
	g := NewGomegaWithT(t)
	id := uint64(1)
	member := &pdpb.Member{Name: "test", MemberId: id}
	membersExist := &MembersInfo{
		Members: []*pdpb.Member{
			member,
		},
		Leader:     member,
		EtcdLeader: member,
	}
	membersExistBytes, err := json.Marshal(membersExist)
	g.Expect(err).NotTo(HaveOccurred())

	membersNotExist := &MembersInfo{
		Members: []*pdpb.Member{},
	}
	membersNotExistBytes, err := json.Marshal(membersNotExist)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName  string
		prePath   string
		preMethod string
		preResp   []byte
		exist     bool
		path      string
		method    string
		want      bool
	}{{
		caseName:  "success_DeleteMemberByID",
		prePath:   fmt.Sprintf("/%s", membersPrefix),
		preMethod: "GET",
		preResp:   membersExistBytes,
		exist:     true,
		path:      fmt.Sprintf("/%s/id/%d", membersPrefix, id),
		method:    "DELETE",
		want:      true,
	}, {
		caseName:  "failed_DeleteMemberByID",
		prePath:   fmt.Sprintf("/%s", membersPrefix),
		preMethod: "GET",
		preResp:   membersExistBytes,
		exist:     true,
		path:      fmt.Sprintf("/%s/id/%d", membersPrefix, id),
		method:    "DELETE",
		want:      false,
	}, {
		caseName:  "delete_not_exit_member",
		prePath:   fmt.Sprintf("/%s", membersPrefix),
		preMethod: "GET",
		preResp:   membersNotExistBytes,
		exist:     false,
		path:      fmt.Sprintf("/%s/id/%d", membersPrefix, id),
		method:    "DELETE",
		want:      true,
	},
	}

	for _, tc := range tcs {
		count := 1
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			if count == 1 {
				g.Expect(request.Method).To(Equal(tc.preMethod), "check method")
				g.Expect(request.URL.Path).To(Equal(tc.prePath), "check url")
				w.Header().Set("Content-Type", ContentTypeJSON)
				w.WriteHeader(http.StatusOK)
				w.Write(tc.preResp)
				count++
				return
			}

			g.Expect(tc.exist).To(BeTrue())
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")
			w.Header().Set("Content-Type", ContentTypeJSON)
			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		err := pdClient.DeleteMemberByID(id)
		if tc.want {
			g.Expect(err).NotTo(HaveOccurred(), "check result")
		} else {
			g.Expect(err).To(HaveOccurred(), "check result")
		}
	}
}

func TestDeleteStore(t *testing.T) {
	g := NewGomegaWithT(t)
	storeID := uint64(1)
	store := &StoreInfo{
		Store:  &MetaStore{Store: &metapb.Store{Id: storeID, State: metapb.StoreState_Up}},
		Status: &StoreStatus{},
	}
	stores := &StoresInfo{
		Count: 1,
		Stores: []*StoreInfo{
			store,
		},
	}

	storesBytes, err := json.Marshal(stores)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName  string
		prePath   string
		preMethod string
		preResp   []byte
		exist     bool
		path      string
		method    string
		want      bool
	}{{
		caseName:  "success_DeleteStore",
		prePath:   fmt.Sprintf("/%s", storesPrefix),
		preMethod: "GET",
		preResp:   storesBytes,
		exist:     true,
		path:      fmt.Sprintf("/%s/%d", storePrefix, storeID),
		method:    "DELETE",
		want:      true,
	}, {
		caseName:  "failed_DeleteStore",
		prePath:   fmt.Sprintf("/%s", storesPrefix),
		preMethod: "GET",
		preResp:   storesBytes,
		exist:     true,
		path:      fmt.Sprintf("/%s/%d", storePrefix, storeID),
		method:    "DELETE",
		want:      false,
	}, {
		caseName:  "delete_not_exist_store",
		prePath:   fmt.Sprintf("/%s", storesPrefix),
		preMethod: "GET",
		preResp:   storesBytes,
		exist:     true,
		path:      fmt.Sprintf("/%s/%d", storePrefix, storeID),
		method:    "DELETE",
		want:      true,
	},
	}

	for _, tc := range tcs {
		count := 1
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			if count == 1 {
				g.Expect(request.Method).To(Equal(tc.preMethod), "check method")
				g.Expect(request.URL.Path).To(Equal(tc.prePath), "check url")
				w.Header().Set("Content-Type", ContentTypeJSON)
				w.WriteHeader(http.StatusOK)
				w.Write(tc.preResp)
				count++
				return
			}

			g.Expect(tc.exist).To(BeTrue())
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			if tc.want {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		defer svc.Close()

		pdClient := NewPDClient(svc.URL, DefaultTimeout, &tls.Config{})
		err := pdClient.DeleteStore(storeID)
		if tc.want {
			g.Expect(err).NotTo(HaveOccurred(), "check result")
		} else {
			g.Expect(err).To(HaveOccurred(), "check result")
		}
	}
}

func readJSON(r io.ReadCloser, data interface{}) error {
	defer r.Close()

	b, err := ioutil.ReadAll(r)
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
		wantMethod  string
		wantPath    string
		wantQuery   string
		checkResult func(t *testing.T, results []reflect.Value)
	}{
		{
			name:   "GetTombStoneStores",
			method: "GetTombStoneStores",
			resp: []byte(`{
	"count": 1,
	"stores": [
		{
			"store": {
			},
			"status": {
			}
		}
	]
}
`),
			statusCode:  http.StatusOK,
			wantMethod:  "GET",
			wantPath:    fmt.Sprintf("/%s", storesPrefix),
			wantQuery:   fmt.Sprintf("state=%d", metapb.StoreState_Tombstone),
			checkResult: checkNoError,
		},
		{
			name:   "UpdateReplicationConfig",
			method: "UpdateReplicationConfig",
			args: []reflect.Value{
				reflect.ValueOf(PDReplicationConfig{}),
			},
			resp:        []byte(``),
			statusCode:  http.StatusOK,
			wantMethod:  "POST",
			wantPath:    fmt.Sprintf("/%s", pdReplicationPrefix),
			checkResult: checkNoError,
		},
		{
			name:   "BeginEvictLeader",
			method: "BeginEvictLeader",
			args: []reflect.Value{
				reflect.ValueOf(uint64(1)),
			},
			statusCode:  http.StatusOK,
			wantMethod:  "POST",
			wantPath:    fmt.Sprintf("/%s", schedulersPrefix),
			checkResult: checkNoError,
		},
		{
			name:   "EndEvictLeader",
			method: "EndEvictLeader",
			args: []reflect.Value{
				reflect.ValueOf(uint64(1)),
			},
			statusCode:  http.StatusNotFound,
			wantMethod:  "DELETE",
			wantPath:    fmt.Sprintf("/%s/evict-leader-scheduler-1", schedulersPrefix),
			checkResult: checkNoError,
		},
		{
			name:   "GetEvictLeaderSchedulers",
			method: "GetEvictLeaderSchedulers",
			resp: []byte(`
[
	"evict-leader-scheduler-1"	
]
`),
			statusCode:  http.StatusOK,
			wantMethod:  "GET",
			wantPath:    fmt.Sprintf("/%s", schedulersPrefix),
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
			wantMethod:  "GET",
			wantPath:    fmt.Sprintf("/%s", pdLeaderPrefix),
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
			wantMethod:  "POST",
			wantPath:    fmt.Sprintf("/%s/%s", pdLeaderTransferPrefix, "foo"),
			checkResult: checkNoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			server := getClientServer(func(w http.ResponseWriter, request *http.Request) {
				g.Expect(request.Method).To(Equal(tt.wantMethod), "check method")
				g.Expect(request.URL.Path).To(Equal(tt.wantPath), "check path")
				g.Expect(request.URL.RawQuery).To(Equal(tt.wantQuery), "check query")

				w.Header().Set("Content-Type", ContentTypeJSON)
				w.WriteHeader(tt.statusCode)
				w.Write(tt.resp)
			})
			defer server.Close()

			pdClient := NewPDClient(server.URL, DefaultTimeout, &tls.Config{})
			args := []reflect.Value{
				reflect.ValueOf(pdClient),
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
