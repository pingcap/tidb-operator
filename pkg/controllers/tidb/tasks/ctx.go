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

	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/tidbapi/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	pdRequestTimeout   = 10 * time.Second
	tidbRequestTimeout = 10 * time.Second
)

type ReconcileContext struct {
	State

	TiDBClient tidbapi.TiDBClient
}

func TaskContextInfoFromPDAndTiDB(state *ReconcileContext, c client.Client, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPDAndTiDB", func(ctx context.Context) task.Result {
		var tlsConfig *tls.Config
		ck := state.Cluster()
		if coreutil.IsTLSClusterEnabled(ck) {
			var err error
			tlsConfig, err = apicall.GetClientTLSConfig(ctx, c, ck)
			if err != nil {
				return task.Fail().With("cannot get tls config from secret: %w", err)
			}
		}
		tidb := state.Object()

		state.TiDBClient = tidbapi.NewTiDBClient(
			coreutil.InstanceAdvertiseURL[scope.TiDB](ck, tidb, coreutil.TiDBStatusPort(tidb)),
			tidbRequestTimeout,
			tlsConfig,
		)
		health, err := state.TiDBClient.GetHealth(ctx)
		if err != nil {
			return task.Complete().With(
				fmt.Sprintf("context without health info is completed, tidb can't be reached: %v", err))
		}
		if health {
			state.SetHealthy()
		}

		return task.Complete().With("get info from tidb")
	})
}
