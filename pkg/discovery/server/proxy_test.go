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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxyServer(t *testing.T) {
	t.Log("create a dashboard server")

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`OK`))
	}))
	defer dashboardServer.Close()

	t.Log("create a proxy server")
	s := NewProxyServer("foo", false)
	proxyToURL, err := url.Parse(dashboardServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	s.(*proxyServer).proxyTo = proxyToURL
	httpServer := httptest.NewServer(s.(*proxyServer))
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code expects %v, got %v", http.StatusOK, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "OK" {
		t.Fatalf("data expects %q, got %q", "OK", string(data))
	}
}

func TestProxyServerTLS(t *testing.T) {
	s := NewProxyServer("foo", true)
	httpServer := httptest.NewServer(s.(*proxyServer))
	defer httpServer.Close()

	// TODO Add tests cases for TLS
}
