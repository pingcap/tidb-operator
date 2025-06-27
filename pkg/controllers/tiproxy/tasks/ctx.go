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

	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/tiproxyapi/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	tlsutil "github.com/pingcap/tidb-operator/pkg/utils/tls"
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
		var (
			scheme    = "http"
			tlsConfig *tls.Config
		)
		ck := state.Cluster()
		if coreutil.IsTLSClusterEnabled(ck) {
			scheme = "https"
			var err error
			tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, c,
				ck.Namespace, coreutil.TLSClusterClientSecretName(ck.Name))
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

		pdc, ok := cm.Get(timanager.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Fail().With("pd client is not registered")
		}
		state.SetPDClient(pdc.Underlay())

		return task.Complete().With("get info from tiproxy")
	})
}
