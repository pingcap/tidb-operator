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
			IP:   "10.16.30.11",
		},
		{
			Name: "vm2",
			IP:   "10.16.30.12",
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

func TestStartETCD(t *testing.T) {
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

	cli := NewClient(Config{
		Addr: ts.URL,
	})

	err := cli.StartETCD()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestStopETCD(t *testing.T) {
	g := NewGomegaWithT(t)

	resp := &api.Response{
		Action:     "stopETCD",
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

	err := cli.StopETCD()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestStartKubelet(t *testing.T) {
	g := NewGomegaWithT(t)

	resp := &api.Response{
		Action:     "startKubelet",
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

	err := cli.StartKubelet()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestStopKubelet(t *testing.T) {
	g := NewGomegaWithT(t)

	resp := &api.Response{
		Action:     "stopKubelet",
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

	err := cli.StopKubelet()
	g.Expect(err).NotTo(HaveOccurred())
}
