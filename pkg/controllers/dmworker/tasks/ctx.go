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

package tasks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State
}

const healthCheckTimeout = 5 * time.Second

func TaskContextHealthFromDMWorker(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ContextHealthFromDMWorker", func(ctx context.Context) task.Result {
		cluster := state.Cluster()
		dw := state.DMWorker()

		addr := coreutil.InstanceAdvertiseAddress[scope.DMWorker](cluster, dw, coreutil.DMWorkerPort(dw))
		scheme := "http"

		var httpClient *http.Client
		if coreutil.IsTLSClusterEnabled(cluster) {
			tlsConfig, err := apicall.GetClientTLSConfig(ctx, c, cluster)
			if err != nil {
				return task.Fail().With("cannot get tls config from secret: %w", err)
			}
			httpClient = &http.Client{
				Timeout: healthCheckTimeout,
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
			}
			scheme = "https"
		} else {
			httpClient = &http.Client{Timeout: healthCheckTimeout}
		}

		url := fmt.Sprintf("%s://%s/status", scheme, addr)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody) //nolint:gosec
		if err != nil {
			return task.Complete().With("cannot build request for dm-worker status: %v", err)
		}

		resp, err := httpClient.Do(req) //nolint:gosec
		if err != nil {
			return task.Complete().With("context without health info is completed, dm-worker can't be reached: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return task.Complete().With("dm-worker returned non-200 status: %d", resp.StatusCode)
		}

		state.SetHealthy()
		return task.Complete().With("dm-worker is healthy")
	})
}
