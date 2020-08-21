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

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/api"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
)

func TestListVMs(t *testing.T) {
	g := NewGomegaWithT(t)

	vms := []*manager.VM{
		{
			Name: "vm1",
			Host: "10.16.30.11",
		},
		{
			Name: "vm2",
			Host: "10.16.30.12",
		},
	}

	resp := &api.Response{
		Action:     "listVMs",
		StatusCode: 200,
		Payload:    vms,
	}

	respJSON, _ := json.Marshal(resp)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(respJSON))
	}))
	defer ts.Close()

	cli := NewClient(Config{
		Addr: ts.URL,
	})

	vms2, err := cli.ListVMs()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vms).To(Equal(vms2))
}

func TestStartVM(t *testing.T) {
	g := NewGomegaWithT(t)

	resp := &api.Response{
		Action:     "startVM",
		StatusCode: 200,
		Message:    "OK",
	}

	respJSON, _ := json.Marshal(resp)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(respJSON))
	}))
	defer ts.Close()

	cli := NewClient(Config{
		Addr: ts.URL,
	})

	err := cli.StartVM(&manager.VM{
		Name: "vm1",
	})
	g.Expect(err).NotTo(HaveOccurred())

	err = cli.StartVM(&manager.VM{
		Host: "127.0.0.1",
	})
	g.Expect(err).To(HaveOccurred())
}

func TestStopVM(t *testing.T) {
	g := NewGomegaWithT(t)

	resp := &api.Response{
		Action:     "stopVM",
		StatusCode: 200,
		Message:    "OK",
	}

	respJSON, _ := json.Marshal(resp)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(respJSON))
	}))
	defer ts.Close()

	cli := NewClient(Config{
		Addr: ts.URL,
	})

	err := cli.StopVM(&manager.VM{
		Name: "vm1",
	})

	g.Expect(err).NotTo(HaveOccurred())

	err = cli.StopVM(&manager.VM{
		Host: "127.0.0.1",
	})
	g.Expect(err).To(HaveOccurred())
}

func TestStartAndStopService(t *testing.T) {
	g := NewGomegaWithT(t)

	resp := &api.Response{
		Action:     "startETCD",
		StatusCode: 200,
		Message:    "OK",
	}

	respJSON, _ := json.Marshal(resp)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(respJSON))
	}))
	defer ts.Close()

	cli := &client{
		cfg: Config{
			Addr: ts.URL,
		},
		httpCli: http.DefaultClient,
	}

	err := cli.startService(manager.ETCDService)
	g.Expect(err).NotTo(HaveOccurred())

	err = cli.stopService(manager.ETCDService)
	g.Expect(err).NotTo(HaveOccurred())
}
