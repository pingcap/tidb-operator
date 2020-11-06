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

package dmapi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
)

const (
	ContentTypeJSON string = "application/json"
)

func getClientServer(h func(http.ResponseWriter, *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(h))
}

func TestGetMembers(t *testing.T) {
	g := NewGomegaWithT(t)
	masters := []*MastersInfo{
		{Name: "dm-master1", MemberID: "1", Alive: false},
		{Name: "dm-master2", MemberID: "2", Alive: true},
		{Name: "dm-master3", MemberID: "3", Alive: true},
	}
	masterResp := MastersResp{
		RespHeader: RespHeader{Result: true, Msg: ""},
		ListMemberResp: []*ListMemberMaster{
			{MembersMaster{
				Msg:     "",
				Masters: masters,
			}},
		},
	}
	masterBytes, err := json.Marshal(masterResp)
	g.Expect(err).NotTo(HaveOccurred())

	workers := []*WorkersInfo{
		{Name: "dm-worker1", Addr: "127.0.0.1:8262", Stage: "free"},
		{Name: "dm-worker2", Addr: "127.0.0.1:8263", Stage: "bound", Source: "mysql-replica-01"},
		{Name: "dm-worker3", Addr: "127.0.0.1:8264", Stage: "offline"},
	}
	workerResp := WorkerResp{
		RespHeader: RespHeader{Result: true, Msg: ""},
		ListMemberResp: []*ListMemberWorker{
			{MembersWorker{
				Msg:     "",
				Workers: workers,
			}},
		},
	}
	workerBytes, err := json.Marshal(workerResp)
	g.Expect(err).NotTo(HaveOccurred())

	leader := MembersLeader{
		Msg:  "",
		Name: "dm-master2",
		Addr: "127.0.0.1:8361",
	}
	leaderResp := LeaderResp{
		RespHeader: RespHeader{Result: true, Msg: ""},
		ListMemberResp: []*ListMemberLeader{
			{leader}},
	}
	leaderBytes, err := json.Marshal(leaderResp)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		path     string
		method   string
		getType  string
		resp     []byte
		want     interface{}
	}{{
		caseName: "GetMasters",
		path:     fmt.Sprintf("/%s", membersPrefix),
		method:   "GET",
		resp:     masterBytes,
		want:     masters,
		getType:  "master",
	}, {
		caseName: "GetWorkers",
		path:     fmt.Sprintf("/%s", membersPrefix),
		method:   "GET",
		resp:     workerBytes,
		want:     workers,
		getType:  "worker",
	}, {
		caseName: "GetLeader",
		path:     fmt.Sprintf("/%s", membersPrefix),
		method:   "GET",
		resp:     leaderBytes,
		want:     leader,
		getType:  "leader",
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(tc.path), "check url")
			g.Expect(request.FormValue(tc.getType)).To(Equal("true"), "check form value")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		var (
			result interface{}
			err    error
		)
		masterClient := NewMasterClient(svc.URL, DefaultTimeout, &tls.Config{}, false)
		switch tc.getType {
		case "master":
			result, err = masterClient.GetMasters()
		case "worker":
			result, err = masterClient.GetWorkers()
		case "leader":
			result, err = masterClient.GetLeader()
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result).To(Equal(tc.want))
	}
}

func TestEvictLeader(t *testing.T) {
	g := NewGomegaWithT(t)
	evictLeaderResp := RespHeader{Result: true, Msg: ""}
	evictLeaderBytes, err := json.Marshal(evictLeaderResp)
	g.Expect(err).NotTo(HaveOccurred())

	svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
		g.Expect(request.Method).To(Equal("PUT"), "check method")
		g.Expect(request.URL.Path).To(Equal(fmt.Sprintf("/%s/1", leaderPrefix)), "check url")

		w.Header().Set("Content-Type", ContentTypeJSON)
		w.Write(evictLeaderBytes)
	})

	masterClient := NewMasterClient(svc.URL, DefaultTimeout, &tls.Config{}, false)
	err = masterClient.EvictLeader()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestDeleteMember(t *testing.T) {
	g := NewGomegaWithT(t)
	deleteMemberResp := RespHeader{Result: true, Msg: ""}
	deleteMemberBytes, err := json.Marshal(deleteMemberResp)
	g.Expect(err).NotTo(HaveOccurred())

	tcs := []struct {
		caseName string
		path     string
		method   string
		resp     []byte
		delType  string
		name     string
	}{{
		caseName: "DeleteMaster",
		path:     fmt.Sprintf("/%s", membersPrefix),
		method:   "DELETE",
		resp:     deleteMemberBytes,
		delType:  "master",
		name:     "dm-master-1",
	}, {
		caseName: "DeleteWorker",
		path:     fmt.Sprintf("/%s", membersPrefix),
		method:   "DELETE",
		resp:     deleteMemberBytes,
		delType:  "worker",
		name:     "dm-worker-1",
	}}

	for _, tc := range tcs {
		svc := getClientServer(func(w http.ResponseWriter, request *http.Request) {
			g.Expect(request.Method).To(Equal(tc.method), "check method")
			g.Expect(request.URL.Path).To(Equal(fmt.Sprintf("%s/%s/%s", tc.path, tc.delType, tc.name)), "check url")

			w.Header().Set("Content-Type", ContentTypeJSON)
			w.Write(tc.resp)
		})
		defer svc.Close()

		masterClient := NewMasterClient(svc.URL, DefaultTimeout, &tls.Config{}, false)
		switch tc.delType {
		case "master":
			err = masterClient.DeleteMaster(tc.name)
		case "worker":
			err = masterClient.DeleteWorker(tc.name)
		}
		g.Expect(err).NotTo(HaveOccurred())
	}
}
