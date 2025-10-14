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

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pd "github.com/tikv/pd/client"
	"github.com/tikv/pd/client/clients/router"
	"github.com/tikv/pd/client/opt"
	"github.com/tikv/pd/client/pkg/caller"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

const (
	defaultGRPCKeepAliveTime    = 10 * time.Second
	defaultGRPCKeepAliveTimeout = 3 * time.Second
	defaultPDServerTimeout      = 3 * time.Second

	defaultLeaderTransferTime = 100 * time.Millisecond
)

func NewPDClient(pdAddrs []string) (pd.Client, error) {
	// init pd-client
	pdCli, err := pd.NewClient(
		caller.Component("tidb-operator"),
		pdAddrs,
		pd.SecurityOption{
			// CAPath:   cfg.Security.ClusterSSLCA,
			// CertPath: cfg.Security.ClusterSSLCert,
			// KeyPath:  cfg.Security.ClusterSSLKey,
		},
		opt.WithGRPCDialOptions(
			grpc.WithKeepaliveParams(
				keepalive.ClientParameters{
					Time:    defaultGRPCKeepAliveTime,
					Timeout: defaultGRPCKeepAliveTimeout,
				},
			),
		),
		opt.WithCustomTimeoutOption(defaultPDServerTimeout),
	)
	if err != nil {
		return nil, err
	}
	return pdCli, nil
}

func PDRegionAccess(ctx context.Context) error {
	var pdEndpoints []string
	// Parse PD endpoints for pd-region action
	if pdEndpointsStr != "" {
		pdEndpoints = strings.Split(pdEndpointsStr, ",")
		for i, endpoint := range pdEndpoints {
			pdEndpoints[i] = strings.TrimSpace(endpoint)
		}
	}
	if len(pdEndpoints) == 0 {
		return fmt.Errorf("pd endpoints not provided")
	}

	var totalCount, failCount atomic.Uint64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(ctx, time.Duration(durationMinutes)*time.Minute)
	defer cancel()
	client, err := NewPDClient(pdEndpoints)
	if err != nil {
		return fmt.Errorf("failed to create PD client: %w", err)
	}
	defer client.Close()

	for i := 1; i <= maxConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					err := accessPDAPI(ctx, client)
					totalCount.Add(1)
					if err != nil &&
						!errors.Is(err, context.DeadlineExceeded) &&
						!errors.Is(err, context.Canceled) &&
						status.Code(err) != codes.Canceled &&
						status.Code(err) != codes.DeadlineExceeded {
						fmt.Printf("[%d-%s] failed to access PD API: %v\n",
							id, time.Now().String(), err,
						)
						failCount.Add(1)
					}
					time.Sleep(time.Duration(sleepInterval) * time.Millisecond)
				}
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("total count: %d, fail count: %d\n", totalCount.Load(), failCount.Load())
	if failCount.Load() > 0 {
		return fmt.Errorf("there are failed PD API accesses")
	}

	return nil
}

func accessPDAPI(ctx context.Context, client pd.Client) error {
	// retry once
	var lastErr error
	for range 3 {
		if lastErr != nil {
			fmt.Printf("[%s] retry because of %v\n", time.Now().String(), lastErr)
		}

		_, _, err1 := client.GetTS(ctx)
		_, err2 := client.BatchScanRegions(ctx, []router.KeyRange{
			{
				StartKey: []byte(""),
				EndKey:   []byte(""),
			},
		}, 1)
		if err1 == nil && err2 == nil {
			return nil
		}

		if err1 != nil {
			lastErr = fmt.Errorf("get ts failed: %w", err1)
			if !strings.Contains(err1.Error(), "not leader") {
				break
			}
		}
		if err2 != nil {
			lastErr = fmt.Errorf("scan region failed: %w", err2)
			// only retry for not leader err
			if !strings.Contains(err2.Error(), "not leader") {
				break
			}
		}
		time.Sleep(defaultLeaderTransferTime)
	}
	if lastErr != nil {
		return lastErr
	}

	return nil
}
