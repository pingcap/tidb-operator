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
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/tiproxyapi/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	pdRequestTimeout      = 10 * time.Second
	tiproxyRequestTimeout = 10 * time.Second
)

type ReconcileContext struct {
	State

	TiProxyClient tiproxyapi.TiProxyClient
}

func TaskContextInfoFromPDAndTiProxy(state *ReconcileContext, c client.Client, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPDAndTiProxy", func(ctx context.Context) task.Result {
		ck := state.Cluster()
		pdc, ok := cm.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Fail().With("pd client is not registered")
		}
		if !pdc.HasSynced() {
			return task.Fail().With("store info is not synced, just wait for next sync")
		}
		state.SetPDClient(pdc.Underlay())

		var (
			scheme    = "http"
			tlsConfig *tls.Config
		)
		if coreutil.IsTiProxyHTTPServerTLSEnabled(ck, state.Object()) {
			scheme = "https"
			var err error
			tlsConfig, err = apicall.GetClientTLSConfig[scope.TiProxy](ctx, c, state.Object())
			if err != nil {
				return task.Fail().With("cannot get tls config from secret: %w", err)
			}
		}
		state.TiProxyClient = tiproxyapi.NewTiProxyClient(TiProxyServiceURL(state.TiProxy(), scheme), tiproxyRequestTimeout, tlsConfig)
		healthy, err := state.TiProxyClient.IsHealthy(ctx)
		if err != nil {
			return task.Complete().With(
				fmt.Sprintf("context without health info is completed, tiproxy can't be reached: %v", err))
		}
		if healthy {
			state.SetHealthy()
		}
		return task.Complete().With("get info from pd and tiproxy")
	})
}
