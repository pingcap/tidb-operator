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
	"time"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/ticdcapi/v1"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	ticdcRequestTimeout = 10 * time.Second
)

type ReconcileContext struct {
	State

	TiCDCClient ticdcapi.TiCDCClient

	Healthy bool

	GracefulWaitTimeInSeconds int64

	// ConfigHash stores the hash of **user-specified** config (i.e.`.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// This ensures that our config overlay logic will not restart the tidb cluster unexpectedly.
	ConfigHash string

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	PodIsTerminating bool
}

func TaskContextInfoFromTiCDC(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ContextInfoFromTiCDC", func(ctx context.Context) task.Result {
		/*var (
			scheme    = "http"
			tlsConfig *tls.Config
		)
		ck := state.Cluster()
		if ck.IsTLSClusterEnabled() {
			scheme = "https"
			var err error
			tlsConfig, err = tlsutil.GetTLSConfigFromSecret(ctx, c,
				ck.Namespace, v1alpha1.TLSClusterClientSecretName(ck.Name))
			if err != nil {
				return task.Fail().With("cannot get tls config from secret: %w", err)
			}
		}
		state.TiDBClient = tidbapi.NewTiDBClient(TiDBServiceURL(state.TiDB(), scheme), ticdcRequestTimeout, tlsConfig)
		health, err := state.TiDBClient.GetHealth(ctx)
		if err != nil {
			return task.Complete().With(
				fmt.Sprintf("context without health info is completed, ticdc can't be reached: %v", err))
		}
		state.Healthy = health*/

		return task.Complete().With("get info from ticdc")
	})
}
