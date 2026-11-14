// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httputil_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/resourcemanagerapi"
	ticdcapi "github.com/pingcap/tidb-operator/v2/pkg/ticdcapi/v1"
	tidbapi "github.com/pingcap/tidb-operator/v2/pkg/tidbapi/v1"
	tiflashapi "github.com/pingcap/tidb-operator/v2/pkg/tiflashapi/v1"
	tikvapi "github.com/pingcap/tidb-operator/v2/pkg/tikvapi/v1"
	tiproxyapi "github.com/pingcap/tidb-operator/v2/pkg/tiproxyapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/tsoapi"
	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
)

// Exercise the public clients, including direct Do calls and TiKV's metrics
// fetch, so a client that bypasses the shared policy cannot replay requests.
func TestComponentClientsRejectRedirects(t *testing.T) {
	ctx := context.Background()
	const timeout = 5 * time.Second
	cases := []struct {
		name   string
		method string
		call   func(string, *tls.Config) error
	}{
		{"PD delete store", http.MethodDelete, func(url string, cfg *tls.Config) error {
			return pdapi.NewPDClient(url, timeout, cfg).DeleteStore(ctx, "1")
		}},
		{"PD cancel delete store", http.MethodPost, func(url string, cfg *tls.Config) error {
			return pdapi.NewPDClient(url, timeout, cfg).CancelDeleteStore(ctx, "1")
		}},
		{"PD delete member", http.MethodDelete, func(url string, cfg *tls.Config) error {
			return pdapi.NewPDClient(url, timeout, cfg).DeleteMember(ctx, "member")
		}},
		{"PD delete member by ID", http.MethodDelete, func(url string, cfg *tls.Config) error {
			client := pdapi.NewPDClient(url, timeout, cfg).(interface {
				DeleteMemberByID(context.Context, uint64) error
			})
			return client.DeleteMemberByID(ctx, 1)
		}},
		{"PD placement rules", http.MethodPost, func(url string, cfg *tls.Config) error {
			return pdapi.NewPDClient(url, timeout, cfg).SetPlacementRulesInBatch(ctx, []pdapi.PlacementRuleOp{{Action: pdapi.PlacementRuleOpAdd}})
		}},
		{"PD evict leader", http.MethodPost, func(url string, cfg *tls.Config) error {
			return pdapi.NewPDClient(url, timeout, cfg).BeginEvictLeader(ctx, "1")
		}},
		{"TSO", http.MethodPost, func(url string, cfg *tls.Config) error {
			return tsoapi.NewTSOClient(url, timeout, cfg).TransferTSOLeader(ctx, "member")
		}},
		{"ResourceManager", http.MethodPost, func(url string, cfg *tls.Config) error {
			return resourcemanagerapi.NewClient(url, timeout, cfg).TransferPrimary(ctx, "member")
		}},
		{"TiDB", http.MethodPost, func(url string, cfg *tls.Config) error {
			return tidbapi.NewTiDBClient(url, timeout, cfg).Activate(ctx, "keyspace")
		}},
		{"TiFlash", http.MethodGet, func(url string, cfg *tls.Config) error {
			_, err := tiflashapi.NewTiFlashClient(url, timeout, cfg).GetStoreStatus(ctx)
			return err
		}},
		{"TiCDC", http.MethodGet, func(url string, cfg *tls.Config) error {
			_, err := ticdcapi.NewTiCDCClient(strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://"), ticdcapi.WithTLS(cfg)).IsHealthy(ctx)
			return err
		}},
		{"TiProxy health", http.MethodGet, func(url string, cfg *tls.Config) error {
			_, err := tiproxyapi.NewTiProxyClient(strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://"), timeout, cfg).IsHealthy(ctx)
			return err
		}},
		{"TiProxy metrics", http.MethodGet, func(url string, cfg *tls.Config) error {
			_, err := tiproxyapi.NewTiProxyClient(strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://"), timeout, cfg).ConnectionCount(ctx)
			return err
		}},
		{"TiProxy labels", http.MethodPut, func(url string, cfg *tls.Config) error {
			return tiproxyapi.NewTiProxyClient(strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://"), timeout, cfg).
				SetLabels(ctx, map[string]string{"region": "us-west-1"})
		}},
		{"TiKV", http.MethodGet, func(url string, cfg *tls.Config) error {
			_, err := tikvapi.NewTiKVClient(url, timeout, cfg, false).GetLeaderCount()
			return err
		}},
	}
	for _, secure := range []bool{false, true} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			for _, tt := range cases {
				t.Run(fmt.Sprintf("TLS=%t/%d/%s", secure, status, tt.name), func(t *testing.T) {
					var targetRequests, targetHandshakes, sourceRequests atomic.Int32
					target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						targetRequests.Add(1)
					}))
					if secure {
						target.TLS = &tls.Config{
							ClientAuth: tls.RequestClientCert,
							GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
								targetHandshakes.Add(1)
								return nil, nil
							},
						}
						target.StartTLS()
					} else {
						target.Start()
					}
					defer target.Close()
					source := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						// Member deletion first discovers the member at the original endpoint.
						if r.URL.Path == "/pd/api/v1/members" {
							_, _ = io.WriteString(w, `{"members":[{"name":"member","member_id":1}]}`)
							return
						}
						// Eviction also reads schedulers after the rejected POST.
						if tt.name == "PD evict leader" && r.Method == http.MethodGet {
							_, _ = io.WriteString(w, `[]`)
							return
						}
						sourceRequests.Add(1)
						assert.Equal(t, tt.method, r.Method)
						if tt.name == "PD placement rules" {
							body, err := io.ReadAll(r.Body)
							assert.NoError(t, err)
							assert.Contains(t, string(body), `"action":"add"`)
						}
						w.Header().Set("Location", target.URL+"/privileged")
						w.WriteHeader(status)
					}))
					var cfg *tls.Config
					if secure {
						source.StartTLS()
						// Model the tenant-controlled skip-verify setting and a configured
						// client certificate; rejection must happen before target TLS.
						cfg = &tls.Config{InsecureSkipVerify: true, Certificates: source.TLS.Certificates} //nolint:gosec
					} else {
						source.Start()
					}
					defer source.Close()

					err := tt.call(source.URL, cfg)
					require.ErrorContains(t, err, "component API redirects are not allowed")
					assert.Equal(t, int32(1), sourceRequests.Load())
					assert.Zero(t, targetRequests.Load(), "redirect target must never receive a request")
					assert.Zero(t, targetHandshakes.Load(), "redirect target must never receive a TLS handshake")
				})
			}
		}
	}
}

func TestRejectRelativeRedirect(t *testing.T) {
	var targetRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/target" {
			targetRequests.Add(1)
			return
		}
		w.Header().Set("Location", "/target")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = httputil.RejectRedirect
	_, err := httputil.GetBodyOK(context.Background(), client, server.URL)
	require.ErrorContains(t, err, "component API redirects are not allowed")
	assert.Zero(t, targetRequests.Load())
}
