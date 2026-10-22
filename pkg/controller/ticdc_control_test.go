// Copyright 2022 PingCAP, Inc.
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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

type ticdcRoundTripper func(*http.Request) (*http.Response, error)

func (f ticdcRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTiCDCMaintenanceOwnerRouting(t *testing.T) {
	for _, otherStatusFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("non-owner status fails=%t", otherStatusFails), func(t *testing.T) {
			testTiCDCMaintenanceOwnerRouting(t, otherStatusFails)
		})
	}
}

func testTiCDCMaintenanceOwnerRouting(t *testing.T, otherStatusFails bool) {
	tc := getTidbCluster()
	cdc := defaultTiCDCControl{}
	a := getCaptureAdvertiseAddressPrefix(tc, 1)
	b := getCaptureAdvertiseAddressPrefix(tc, 0)
	// The cached owner is b, but b has already resigned to a.
	tc.Status.TiCDC.Captures = map[string]v1alpha1.TiCDCCapture{
		TiCDCMemberName(tc.Name) + "-0": {ID: "b", Ready: true, IsOwner: true},
		TiCDCMemberName(tc.Name) + "-1": {ID: "a", Ready: true},
	}
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	var queries, drains int
	http.DefaultTransport = ticdcRoundTripper(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.Path {
		case "/status":
			if otherStatusFails && r.URL.Hostname() == b {
				return nil, fmt.Errorf("lookup %s: no such host", b)
			}
			body = fmt.Sprintf(`{"id":%q,"is_owner":%t}`, r.URL.Hostname(), r.URL.Hostname() == a)
		case "/api/v1/captures":
			queries++
			if r.URL.Hostname() != a {
				t.Errorf("captures query sent to %s, want new owner %s", r.URL.Hostname(), a)
			}
			payload, _ := json.Marshal([]captureInfo{
				// The response address must not override the discovered owner Pod URL.
				{ID: "a", AdvertiseAddr: "unreachable.invalid:8301", IsOwner: true},
				{ID: "b", AdvertiseAddr: b + ":8301"},
			})
			body = string(payload)
		case "/api/v1/captures/drain":
			drains++
			if r.URL.Hostname() != a {
				t.Errorf("drain sent to %s, want new owner %s", r.URL.Hostname(), a)
			}
			var payload drainCaptureRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.CaptureID != "b" {
				t.Errorf("drain must target b: %+v, %v", payload, err)
			}
			body = `{"current_table_count":0}`
		default:
			t.Errorf("unexpected request, must not resign new owner: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	if resigned, err := cdc.ResignOwner(tc, 0); err != nil || !resigned {
		t.Fatalf("b has already resigned: %v, %v", resigned, err)
	}
	if count, retry, err := cdc.DrainCapture(tc, 0); err != nil || retry || count != 0 {
		t.Fatalf("drain failed: %d, %v, %v", count, retry, err)
	}
	if queries != 2 || drains != 1 {
		t.Fatalf("unexpected maintenance requests: queries=%d drains=%d", queries, drains)
	}
}

func TestTiCDCOwnerDiscoveryFailure(t *testing.T) {
	for _, mode := range []string{"no owner", "two owners", "owner DNS failure"} {
		t.Run(mode, func(t *testing.T) {
			tc := getTidbCluster()
			cdc := defaultTiCDCControl{}
			a := getCaptureAdvertiseAddressPrefix(tc, 1)
			tc.Status.TiCDC.Captures = map[string]v1alpha1.TiCDCCapture{
				TiCDCMemberName(tc.Name) + "-0": {ID: "b", Ready: true, IsOwner: true},
				TiCDCMemberName(tc.Name) + "-1": {ID: "a", Ready: true},
			}
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			http.DefaultTransport = ticdcRoundTripper(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/status" {
					t.Fatalf("must wait for owner discovery before maintenance: %s", r.URL.Path)
				}
				if mode == "owner DNS failure" && r.URL.Hostname() == a {
					return nil, fmt.Errorf("lookup %s: no such host", a)
				}
				body := fmt.Sprintf(`{"id":%q,"is_owner":%t}`, r.URL.Hostname(), mode == "two owners")
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			if ok, err := cdc.ResignOwner(tc, 0); err == nil || ok {
				t.Fatalf("must retry resign: ok=%v err=%v", ok, err)
			}
			if _, retry, err := cdc.DrainCapture(tc, 0); err == nil && !retry {
				t.Fatal("must retry drain")
			}
		})
	}
}

func TestTiCDCGetCapturesHTTPStatusHandling(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer svr.Close()
			captures, retry, err := getCaptures(svr.Client(), svr.URL)
			if err != nil || len(captures) != 0 || retry != (code == http.StatusServiceUnavailable) {
				t.Fatalf("unexpected HTTP status handling: %v, %v, %v", captures, retry, err)
			}
		})
	}
}

func TestTiCDCSingleCaptureSkipsMaintenance(t *testing.T) {
	tc := getTidbCluster()
	tc.Spec.TiCDC = &v1alpha1.TiCDCSpec{Replicas: 2}
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status":
			fmt.Fprint(w, `{"id":"b","is_owner":true}`)
		case "/api/v1/captures":
			json.NewEncoder(w).Encode([]captureInfo{{ID: "b", AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 0), IsOwner: true}})
		default:
			t.Errorf("single capture must skip resign and drain requests: %s", r.URL.Path)
		}
	}))
	defer svr.Close()
	cdc := defaultTiCDCControl{testURL: svr.URL}
	if ok, err := cdc.ResignOwner(tc, 0); !ok || err != nil {
		t.Fatalf("expected resign to be skipped, got %v, %v", ok, err)
	}
	if count, retry, err := cdc.DrainCapture(tc, 0); count != 0 || retry || err != nil {
		t.Fatalf("expected drain to be skipped, got %d, %v, %v", count, retry, err)
	}
}

func TestTiCDCGetCapturesInvalidResponse(t *testing.T) {
	for _, tt := range []struct {
		name string
		code int
		body string
	}{
		{"internal error", 500, `{"error":"no such host"}`},
		{"error with empty list", 500, `[]`},
		{"invalid json", 200, `broken`},
		{"empty list", 200, `[]`},
		{"null", 200, `null`},
		{"missing capture identity", 200, `[{}]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
				fmt.Fprint(w, tt.body)
			}))
			defer svr.Close()
			_, retry, err := getCaptures(svr.Client(), svr.URL)
			if err == nil && !retry {
				t.Fatal("failed capture query must prevent maintenance from proceeding")
			}
		})
	}
}

func TestTiCDCDrainInvalidResponse(t *testing.T) {
	for _, tt := range []struct {
		name string
		code int
		body string
	}{
		{"internal error", 500, `{"current_table_count":0}`},
		{"invalid json", 200, `broken`},
		{"missing count", 200, `{}`},
		{"null count", 200, `{"current_table_count":null}`},
		{"negative count", 200, `{"current_table_count":-1}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tc := getTidbCluster()
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/status":
					fmt.Fprint(w, `{"id":"owner","is_owner":true}`)
				case "/api/v1/captures":
					json.NewEncoder(w).Encode([]captureInfo{
						{ID: "target", AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1)},
						{ID: "owner", AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 0), IsOwner: true},
					})
				case "/api/v1/captures/drain":
					w.WriteHeader(tt.code)
					fmt.Fprint(w, tt.body)
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
				}
			}))
			defer svr.Close()
			cdc := defaultTiCDCControl{testURL: svr.URL}
			_, retry, err := cdc.DrainCapture(tc, 1)
			if err == nil && !retry {
				t.Fatal("invalid drain response must not report completion")
			}
		})
	}
}

func TestTiCDCControllerResignOwner(t *testing.T) {
	g := NewGomegaWithT(t)

	cdc := defaultTiCDCControl{}
	tc := getTidbCluster()

	cases := []struct {
		caseName    string
		handlers    map[string]func(http.ResponseWriter, *http.Request)
		ordinal     int32
		expectedOk  types.GomegaMatcher
		expectedErr types.GomegaMatcher
	}{
		{
			caseName: "1 captures",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
						IsOwner:       true,
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:     1,
			expectedOk:  BeTrue(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, no owner",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:     1,
			expectedOk:  BeFalse(),
			expectedErr: Not(BeNil()),
		},
		{
			caseName: "2 captures, resign owner ok",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/owner/resign": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusAccepted)
				},
			},
			ordinal:     1,
			expectedOk:  BeFalse(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, resign owner 404",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/owner/resign": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
			},
			ordinal:     1,
			expectedOk:  BeTrue(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, resign owner 503",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/owner/resign": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				},
			},
			ordinal:     1,
			expectedOk:  BeFalse(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, get captures 503",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				},
			},
			ordinal:     1,
			expectedOk:  BeFalse(),
			expectedErr: BeNil(),
		},
	}

	for _, c := range cases {
		mux := http.NewServeMux()
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"id":"owner","is_owner":true}`)
		})
		svr := httptest.NewServer(mux)
		for p, h := range c.handlers {
			mux.HandleFunc(p, h)
		}
		cdc.testURL = svr.URL
		ok, err := cdc.ResignOwner(tc, c.ordinal)
		g.Expect(ok).Should(c.expectedOk, c.caseName)
		g.Expect(err).Should(c.expectedErr, c.caseName)
		svr.Close()
	}
}

func TestTiCDCControllerDrainCaptureMultiClusters(t *testing.T) {
	g := NewGomegaWithT(t)
	cdc := defaultTiCDCControl{}
	tcCd1 := getTidbClusterWithClusterDomain("cluster.1")
	tcCd2 := getTidbClusterWithClusterDomain("cluster.2")

	handlers := map[string]func(http.ResponseWriter, *http.Request){
		"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
			cp := []captureInfo{{
				ID:            "cluster-1-capture-server",
				AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tcCd1, 1),
				IsOwner:       true,
			}, {
				ID:            "cluster-2-capture-server",
				AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tcCd2, 1),
				IsOwner:       false,
			}}
			payload, err := json.Marshal(cp)
			g.Expect(err).Should(BeNil())
			fmt.Fprint(w, string(payload))
		},
		"/api/v1/captures/drain": func(w http.ResponseWriter, req *http.Request) {
			body, err := io.ReadAll(req.Body)
			g.Expect(err).Should(BeNil())
			var reqPayload drainCaptureRequest
			err = json.Unmarshal(body, &reqPayload)
			g.Expect(err).Should(BeNil())
			g.Expect(reqPayload.CaptureID).Should(
				Equal("cluster-2-capture-server"),
			)

			payload, err := json.Marshal(
				drainCaptureResp{CurrentTableCount: 1},
			)
			g.Expect(err).Should(BeNil())
			fmt.Fprint(w, string(payload))
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"owner","is_owner":true}`)
	})
	svr := httptest.NewServer(mux)
	for p, h := range handlers {
		mux.HandleFunc(p, h)
	}
	cdc.testURL = svr.URL
	count, retry, err := cdc.DrainCapture(tcCd2, 1)
	g.Expect(count).Should(Equal(1))
	g.Expect(err).Should(BeNil())
	g.Expect(retry).Should(BeFalse())
	svr.Close()
}

func TestTiCDCControllerDrainCapture(t *testing.T) {
	g := NewGomegaWithT(t)

	cdc := defaultTiCDCControl{}
	tc := getTidbCluster()

	cases := []struct {
		caseName      string
		handlers      map[string]func(http.ResponseWriter, *http.Request)
		ordinal       int32
		expectedCount types.GomegaMatcher
		expectedErr   types.GomegaMatcher
		expectedRetry types.GomegaMatcher
	}{
		{
			caseName: "1 captures",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
						IsOwner:       true,
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:       1,
			expectedCount: BeZero(),
			expectedErr:   BeNil(),
			expectedRetry: BeFalse(),
		},
		{
			caseName: "2 captures, no self",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:       3,
			expectedCount: BeZero(),
			expectedErr:   Not(BeNil()),
			expectedRetry: BeFalse(),
		},
		{
			caseName: "2 captures, no owner",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:       1,
			expectedCount: BeZero(),
			expectedErr:   Not(BeNil()),
			expectedRetry: BeFalse(),
		},
		{
			caseName: "2 captures, drain capture ok 0",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/captures/drain": func(w http.ResponseWriter, req *http.Request) {
					payload, err := json.Marshal(drainCaptureResp{CurrentTableCount: 0})
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:       1,
			expectedCount: BeZero(),
			expectedErr:   BeNil(),
			expectedRetry: BeFalse(),
		},
		{
			caseName: "2 captures, drain capture ok 1",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/captures/drain": func(w http.ResponseWriter, req *http.Request) {
					body, err := io.ReadAll(req.Body)
					g.Expect(err).Should(BeNil())
					var reqPayload drainCaptureRequest
					err = json.Unmarshal(body, &reqPayload)
					g.Expect(err).Should(BeNil())
					g.Expect(reqPayload.CaptureID).Should(Equal("1"))

					payload, err := json.Marshal(drainCaptureResp{CurrentTableCount: 1})
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:       1,
			expectedCount: Equal(1),
			expectedErr:   BeNil(),
			expectedRetry: BeFalse(),
		},
		{
			caseName: "2 captures, drain capture 404",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/captures/drain": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
			},
			ordinal:       1,
			expectedCount: BeZero(),
			expectedErr:   BeNil(),
			expectedRetry: BeFalse(),
		},
		{
			caseName: "2 captures, drain capture 503",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/captures/drain": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				},
			},
			ordinal:       1,
			expectedCount: BeZero(),
			expectedErr:   BeNil(),
			expectedRetry: BeTrue(),
		},
		{
			caseName: "2 captures, get captures 503",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				},
			},
			ordinal:       1,
			expectedCount: BeZero(),
			expectedErr:   BeNil(),
			expectedRetry: BeTrue(),
		},
	}

	for _, c := range cases {
		mux := http.NewServeMux()
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"id":"owner","is_owner":true}`)
		})
		svr := httptest.NewServer(mux)
		for p, h := range c.handlers {
			mux.HandleFunc(p, h)
		}
		cdc.testURL = svr.URL
		count, retry, err := cdc.DrainCapture(tc, c.ordinal)
		g.Expect(count).Should(c.expectedCount, c.caseName)
		g.Expect(err).Should(c.expectedErr, c.caseName)
		g.Expect(retry).Should(c.expectedRetry, c.caseName)
		svr.Close()
	}
}

func TestTiCDCControllerIsHealthy(t *testing.T) {
	g := NewGomegaWithT(t)

	cdc := defaultTiCDCControl{}
	tc := getTidbCluster()

	cases := []struct {
		caseName    string
		handlers    map[string]func(http.ResponseWriter, *http.Request)
		ordinal     int32
		expectedOk  types.GomegaMatcher
		expectedErr types.GomegaMatcher
	}{
		{
			caseName: "1 captures, healthy",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						AdvertiseAddr: req.Host,
						IsOwner:       true,
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/health": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
			ordinal:     1,
			expectedOk:  BeTrue(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, no owner, unhealthy",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 1),
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
			},
			ordinal:     1,
			expectedOk:  BeFalse(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, healthy",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: req.Host,
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/health": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			},
			ordinal:     1,
			expectedOk:  BeTrue(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, health 404",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: req.Host,
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/health": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
			},
			ordinal:     1,
			expectedOk:  BeTrue(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, unhealthy, 500",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					cp := []captureInfo{{
						ID:            "1",
						IsOwner:       true,
						AdvertiseAddr: req.Host,
					}, {
						ID:            "2",
						AdvertiseAddr: getCaptureAdvertiseAddressPrefix(tc, 2),
					}}
					payload, err := json.Marshal(cp)
					g.Expect(err).Should(BeNil())
					fmt.Fprint(w, string(payload))
				},
				"/api/v1/health": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
			ordinal:     1,
			expectedOk:  BeFalse(),
			expectedErr: BeNil(),
		},
		{
			caseName: "2 captures, get captures 503",
			handlers: map[string]func(http.ResponseWriter, *http.Request){
				"/api/v1/captures": func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				},
			},
			ordinal:     1,
			expectedOk:  BeFalse(),
			expectedErr: BeNil(),
		},
	}

	for _, c := range cases {
		mux := http.NewServeMux()
		svr := httptest.NewServer(mux)
		for p, h := range c.handlers {
			mux.HandleFunc(p, h)
		}
		cdc.testURL = svr.URL
		ok, err := cdc.IsHealthy(tc, c.ordinal)
		g.Expect(ok).Should(c.expectedOk, c.caseName)
		g.Expect(err).Should(c.expectedErr, c.caseName)
		svr.Close()
	}
}
