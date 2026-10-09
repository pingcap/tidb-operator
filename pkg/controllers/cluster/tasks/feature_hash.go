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
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

// TaskFeatureGatesHash persists the adopted definitions before reconciliation
// changes finalizers, propagates features, or updates managed resources.
type TaskFeatureGatesHash struct {
	Client client.Client
}

func NewTaskFeatureGatesHash(c client.Client) task.Task[ReconcileContext] {
	return &TaskFeatureGatesHash{Client: c}
}

func (*TaskFeatureGatesHash) Name() string { return "FeatureGatesHash" }

func (t *TaskFeatureGatesHash) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()
	hash := features.CurrentFeatureGateDefinitionHash
	if rtx.Cluster.Status.FeatureGatesHash == hash {
		return task.Complete().With("feature gate hash is recorded")
	}
	// Do not publish the new hash in the reconcile context before persistence.
	updated := rtx.Cluster.DeepCopy()
	updated.Status.FeatureGatesHash = hash
	if err := t.Client.Status().Update(ctx, updated); err != nil {
		return task.Fail().With("can't record feature gate hash: %w", err)
	}
	// An outdated CRD can prune unknown status fields even on successful writes.
	if updated.Status.FeatureGatesHash != hash {
		return task.Fail().With("feature gate hash was not persisted; check the Cluster CRD schema")
	}
	rtx.Cluster = updated
	return task.Complete().With("feature gate hash is recorded")
}
