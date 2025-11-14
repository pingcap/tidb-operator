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

	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/ticdcapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State

	TiCDCClient ticdcapi.TiCDCClient

	MemberID string
	IsOwner  bool
}

func TaskContextInfoFromTiCDC(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ContextInfoFromTiCDC", func(ctx context.Context) task.Result {
		var tlsConfig *tls.Config
		ck := state.Cluster()
		if coreutil.IsTLSClusterEnabled(ck) {
			var err error
			tlsConfig, err = apicall.GetClientTLSConfig[scope.TiCDC](ctx, c, state.Object())
			if err != nil {
				return task.Fail().With("cannot get tls config from secret: %w", err)
			}
		}
		// TODO(liubo02): cache client
		state.TiCDCClient = ticdcapi.NewTiCDCClient(coreutil.TiCDCAdvertiseURL(state.TiCDC()),
			ticdcapi.WithTLS(tlsConfig),
			// TODO(liubo02): fix it
			ticdcapi.DisableKeepAlives(),
		)
		healthy, err := state.TiCDCClient.IsHealthy(ctx)
		if err != nil {
			return task.Complete().With(
				fmt.Sprintf("context without health info is completed, ticdc can't be reached: %v", err))
		}
		if healthy {
			state.SetHealthy()
		}

		status, err := state.TiCDCClient.GetStatus(ctx)
		if err != nil {
			return task.Fail().With("failed to get status from TiCDC: %w", err)
		}
		state.MemberID = status.ID
		state.IsOwner = status.IsOwner

		return task.Complete().With("get info from ticdc")
	})
}
