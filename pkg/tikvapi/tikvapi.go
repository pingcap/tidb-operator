// Copyright 2021 PingCAP, Inc.
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

package tikvapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/pingcap/errors"
	logbackup "github.com/pingcap/kvproto/pkg/logbackuppb"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"
)

const (
	DefaultTimeout        = 5 * time.Second
	metricNameRegionCount = "tikv_raftstore_region_count"
	labelNameLeaderCount  = "leader"
	metricsPrefix         = "metrics"
)

// TiKVClient provides tikv server's api
type TiKVClient interface {
	GetLeaderCount() (int, error)
	FlushLogBackupTasks(ctx context.Context) error
}

type lazyGRPCConn struct {
	target string
	opts   []grpc.DialOption

	initMu sync.Mutex
	cached *grpc.ClientConn
}

func (l *lazyGRPCConn) conn() (*grpc.ClientConn, error) {
	l.initMu.Lock()
	defer l.initMu.Unlock()

	if l.cached != nil && l.cached.GetState() != connectivity.Shutdown {
		return l.cached, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	ch, err := grpc.DialContext(ctx, l.target, l.opts...)
	if err != nil {
		return nil, errors.Annotatef(err, "during connecting to %s", l.target)
	}

	l.cached = ch
	return ch, nil
}

// tikvClient is default implementation of TiKVClient
type tikvClient struct {
	url           string
	httpClient    *http.Client
	grpcConnector *lazyGRPCConn
}

// FlushLogBackupTasks implements TiKVClient.
func (c *tikvClient) FlushLogBackupTasks(ctx context.Context) error {
	conn, err := c.grpcConnector.conn()
	if err != nil {
		return err
	}
	cli := logbackup.NewLogBackupClient(conn)
	res, err := cli.FlushNow(ctx, &logbackup.FlushNowRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return err
	}

	for _, r := range res.Results {
		if !r.Success {
			return errors.Errorf("force flush failed for task %s: %s", r.TaskName, r.ErrorMessage)
		}
	}
	return nil
}

// GetLeaderCount gets region leader count from the URL
func (c *tikvClient) GetLeaderCount() (int, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, metricsPrefix)
	transport := c.httpClient.Transport
	mfChan := make(chan *dto.MetricFamily, 1024)

	go func() {
		if err := prom2json.FetchMetricFamilies(apiURL, mfChan, transport); err != nil {
			klog.Errorf("Fail to get region leader count from %s, error: %v", apiURL, err)
		}
	}()

	fms := []*prom2json.Family{}
	for mfc := range mfChan {
		fm := prom2json.NewFamily(mfc)
		fms = append(fms, fm)
	}
	for _, fm := range fms {
		if fm.Name == metricNameRegionCount {
			for _, m := range fm.Metrics {
				if m, ok := m.(prom2json.Metric); ok && m.Labels["type"] == labelNameLeaderCount {
					return strconv.Atoi(m.Value)
				}
			}
		}
	}

	return 0, fmt.Errorf("metric %s{type=\"%s\"} not found for %s", metricNameRegionCount, labelNameLeaderCount, apiURL)
}

type TiKVClientOpts struct {
	HTTPEndpoint      string
	GRPCEndpoint      string
	Timeout           time.Duration
	TLSConfig         *tls.Config
	DisableKeepAlives bool
}

// NewTiKVClient returns a new TiKVClient
func NewTiKVClient(opts TiKVClientOpts) TiKVClient {
	return &tikvClient{
		url: opts.HTTPEndpoint,
		grpcConnector: &lazyGRPCConn{
			target: opts.GRPCEndpoint,
			opts: []grpc.DialOption{
				grpc.WithTransportCredentials(credentials.NewTLS(opts.TLSConfig)),
			},
		},
		httpClient: &http.Client{
			Timeout: opts.Timeout,
			Transport: &http.Transport{
				TLSClientConfig:       opts.TLSConfig,
				DisableKeepAlives:     opts.DisableKeepAlives,
				ResponseHeaderTimeout: 10 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).DialContext,
			},
		},
	}
}
