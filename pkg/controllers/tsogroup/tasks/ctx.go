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

	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	tsom "github.com/pingcap/tidb-operator/v2/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type ReconcileContext struct {
	State
}

func TaskContextPDMSProtection(state *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ContextPDMSProtection", func(ctx context.Context) task.Result {
		tg := state.Object()
		ns, cluster := tg.Namespace, tg.Spec.Cluster.Name

		pdgs, err := apicall.ListGroups[scope.PDGroup](ctx, c, ns, cluster)
		if err != nil {
			return task.Fail().With("cannot list PDGroups: %w", err)
		}
		pds, err := apicall.ListClusterInstances[scope.PD](ctx, c, ns, cluster)
		if err != nil {
			return task.Fail().With("cannot list PDs: %w", err)
		}
		tgs, err := apicall.ListGroups[scope.TSOGroup](ctx, c, ns, cluster)
		if err != nil {
			return task.Fail().With("cannot list TSOGroups: %w", err)
		}

		state.SetPDMSProtectionContext(pdgs, pds, tgs)
		return task.Complete().With("PDMS protection context is set")
	})
}

func TaskContextTSOClient(state *ReconcileContext, m tsom.TSOClientManager) task.Task {
	return task.NameTaskFunc("ContextTSOClient", func(_ context.Context) task.Result {
		if err := m.Register(state.Object()); err != nil {
			return task.Fail().With("cannot register tso client: %v", err)
		}
		return task.Complete().With("tso client is registered")
	})
}
