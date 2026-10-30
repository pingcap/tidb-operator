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
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/compatibility"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	tidbapi "github.com/pingcap/tidb-operator/v2/pkg/tidbapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	smoothUpgradeRequestTimeout = 10 * time.Second
	smoothUpgradeRetryInterval  = 10 * time.Second
)

// tidbClientFactory creates a TiDB HTTP client for the given instance.
// Accepting this as a parameter enables test injection without changing task semantics.
type tidbClientFactory func(ctx context.Context, c client.Client, ck *v1alpha1.Cluster, tidb *v1alpha1.TiDB) (tidbapi.TiDBClient, error)

// TaskSmoothUpgradeStart calls /upgrade/start on a healthy TiDB instance before rolling upgrade begins.
// It is a no-op when the change is not a version upgrade, or when either the source or target version
// does not support smooth upgrade (< v7.5.0).
func TaskSmoothUpgradeStart(state *ReconcileContext, c client.Client) task.Task {
	return taskSmoothUpgradeStart(state, c, newTiDBClientForGroup)
}

func taskSmoothUpgradeStart(state *ReconcileContext, c client.Client, factory tidbClientFactory) task.Task {
	return task.NameTaskFunc("SmoothUpgradeStart", func(ctx context.Context) task.Result {
		dbg := state.TiDBGroup()

		if !needVersionUpgrade(dbg) {
			return task.Complete().With("not a version upgrade, skipping smooth upgrade start")
		}
		if !compatibility.SupportsSmoothUpgrade(dbg.Status.Version) ||
			!compatibility.SupportsSmoothUpgrade(dbg.Spec.Template.Spec.Version) {
			return task.Complete().With("version does not support smooth upgrade, skipping")
		}
		if dbg.Annotations[v1alpha1.AnnoKeySmoothUpgradePhase] == v1alpha1.AnnoValSmoothUpgradePhaseInProgress {
			return task.Complete().With("smooth upgrade already started")
		}

		tidb := pickReadyTiDB(state.TiDBSlice())
		if tidb == nil {
			return task.Retry(smoothUpgradeRetryInterval).With("no ready TiDB instance available for upgrade/start")
		}

		tidbClient, err := factory(ctx, c, state.Cluster(), tidb)
		if err != nil {
			return task.Retry(smoothUpgradeRetryInterval).With("cannot create TiDB client for upgrade/start: %v", err)
		}

		if err := tidbClient.UpgradeStart(ctx, dbg.Spec.Template.Spec.Keyspace); err != nil {
			return task.Retry(smoothUpgradeRetryInterval).With("upgrade/start failed, will retry: %v", err)
		}

		phase := v1alpha1.AnnoValSmoothUpgradePhaseInProgress
		if err := patchSmoothUpgradeAnnotation(ctx, c, dbg, &phase); err != nil {
			return task.Fail().With("failed to set smooth upgrade annotation: %w", err)
		}

		return task.Complete().With("smooth upgrade started, DDL paused")
	})
}

// TaskSmoothUpgradeFinish calls /upgrade/finish on a healthy TiDB instance after all pods are upgraded.
// It must run after TaskStatusRevisionAndReplicas so that dbg.Status.Version reflects the new version,
// making needVersionUpgrade() return false as the "all done" signal.
func TaskSmoothUpgradeFinish(state *ReconcileContext, c client.Client) task.Task {
	return taskSmoothUpgradeFinish(state, c, newTiDBClientForGroup)
}

func taskSmoothUpgradeFinish(state *ReconcileContext, c client.Client, factory tidbClientFactory) task.Task {
	return task.NameTaskFunc("SmoothUpgradeFinish", func(ctx context.Context) task.Result {
		dbg := state.TiDBGroup()

		if dbg.Annotations[v1alpha1.AnnoKeySmoothUpgradePhase] != v1alpha1.AnnoValSmoothUpgradePhaseInProgress {
			return task.Complete().With("no smooth upgrade in progress")
		}
		if needVersionUpgrade(dbg) {
			return task.Complete().With("upgrade still in progress, finish not yet")
		}

		tidb := pickReadyTiDB(state.TiDBSlice())
		if tidb == nil {
			return task.Retry(smoothUpgradeRetryInterval).With("no ready TiDB instance available for upgrade/finish")
		}

		tidbClient, err := factory(ctx, c, state.Cluster(), tidb)
		if err != nil {
			return task.Retry(smoothUpgradeRetryInterval).With("cannot create TiDB client for upgrade/finish: %v", err)
		}

		if err := tidbClient.UpgradeFinish(ctx); err != nil {
			return task.Retry(smoothUpgradeRetryInterval).With("upgrade/finish failed, will retry: %v", err)
		}

		if err := patchSmoothUpgradeAnnotation(ctx, c, dbg, nil); err != nil {
			return task.Fail().With("failed to remove smooth upgrade annotation: %w", err)
		}

		return task.Complete().With("smooth upgrade finished, DDL resumed")
	})
}

// pickReadyTiDB returns the first TiDB instance that is in the Ready state.
func pickReadyTiDB(dbs []*v1alpha1.TiDB) *v1alpha1.TiDB {
	for _, db := range dbs {
		if coreutil.IsReady[scope.TiDB](db) {
			return db
		}
	}
	return nil
}

// newTiDBClientForGroup creates a TiDB HTTP client targeting the given TiDB instance.
func newTiDBClientForGroup(ctx context.Context, c client.Client, ck *v1alpha1.Cluster, tidb *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
	url := coreutil.InstanceAdvertiseURL[scope.TiDB](ck, tidb, coreutil.TiDBStatusPort(tidb))
	if !coreutil.IsTLSClusterEnabled(ck) {
		return tidbapi.NewTiDBClient(url, smoothUpgradeRequestTimeout, nil), nil
	}
	tlsConfig, err := apicall.GetClientTLSConfig(ctx, c, ck)
	if err != nil {
		return nil, fmt.Errorf("cannot get TLS config: %w", err)
	}
	return tidbapi.NewTiDBClient(url, smoothUpgradeRequestTimeout, tlsConfig), nil
}

type annotationPatch struct {
	Metadata annotationPatchMetadata `json:"metadata"`
}

type annotationPatchMetadata struct {
	ResourceVersion string             `json:"resourceVersion"`
	Annotations     map[string]*string `json:"annotations"`
}

// patchSmoothUpgradeAnnotation sets (value non-nil) or deletes (value nil) the smooth upgrade annotation.
func patchSmoothUpgradeAnnotation(ctx context.Context, c client.Client, dbg *v1alpha1.TiDBGroup, value *string) error {
	p := annotationPatch{
		Metadata: annotationPatchMetadata{
			ResourceVersion: dbg.GetResourceVersion(),
			Annotations: map[string]*string{
				v1alpha1.AnnoKeySmoothUpgradePhase: value,
			},
		},
	}
	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}
	if err := c.Patch(ctx, dbg, client.RawPatch(types.MergePatchType, data)); err != nil {
		return fmt.Errorf("cannot patch smooth upgrade annotation on %s/%s: %w", dbg.Namespace, dbg.Name, err)
	}
	return nil
}
