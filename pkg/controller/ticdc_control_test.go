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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

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
			expectedOk:  BeTrue(),
			expectedErr: BeNil(),
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
