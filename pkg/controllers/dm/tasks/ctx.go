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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State

	MemberID string
}

func TaskContextInfoFromDM(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ContextInfoFromDM", func(ctx context.Context) task.Result {
		cluster := state.Cluster()
		dm := state.DM()

		addr := coreutil.InstanceAdvertiseAddress[scope.DM](cluster, dm, coreutil.DMPort(dm))
		scheme := "http"

		var httpClient *http.Client
		if coreutil.IsTLSClusterEnabled(cluster) {
			tlsConfig, err := apicall.GetClientTLSConfig(ctx, c, cluster)
			if err != nil {
				return task.Fail().With("cannot get tls config from secret: %w", err)
			}
			httpClient = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
			}
			scheme = "https"
		} else {
			httpClient = http.DefaultClient
		}

		url := fmt.Sprintf("%s://%s/api/v1/cluster/info", scheme, addr)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody) //nolint:gosec
		if err != nil {
			return task.Complete().With("cannot build request for dm-master cluster info: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return task.Complete().With("context without health info is completed, dm-master can't be reached: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return task.Complete().With("dm-master returned non-200 status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return task.Complete().With("cannot read dm-master cluster info response: %v", err)
		}

		var info struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &info); err != nil {
			return task.Complete().With("cannot parse dm-master cluster info response: %v", err)
		}

		state.SetHealthy()
		state.MemberID = info.Name

		return task.Complete().With("get info from dm-master")
	})
}
