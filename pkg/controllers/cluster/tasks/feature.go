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
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

type TaskFeatureGates struct {
	Logger logr.Logger
	Client client.Client
}

func NewTaskFeatureGates(logger logr.Logger, c client.Client) task.Task[ReconcileContext] {
	return &TaskFeatureGates{
		Logger: logger,
		Client: c,
	}
}

func (*TaskFeatureGates) Name() string {
	return "FeatureGates"
}

//nolint:gocyclo // refactor if possible
func (t *TaskFeatureGates) Sync(ctx task.Context[ReconcileContext]) task.Result {
	rtx := ctx.Self()

	fs := coreutil.EnabledFeatures(rtx.Cluster)

	for _, pdg := range rtx.PDGroups {
		if err := patchFeatures[scope.PDGroup](ctx, t.Client, pdg, fs); err != nil {
			return task.Fail().With("can't update feature gates for pd group %s/%s: %w", pdg.Namespace, pdg.Name, err)
		}
	}
	for _, tg := range rtx.TSOGroups {
		if err := patchFeatures[scope.TSOGroup](ctx, t.Client, tg, fs); err != nil {
			return task.Fail().With("can't update feature gates for tso group %s/%s: %w", tg.Namespace, tg.Name, err)
		}
	}
	for _, sg := range rtx.SchedulingGroups {
		if err := patchFeatures[scope.SchedulingGroup](ctx, t.Client, sg, fs); err != nil {
			return task.Fail().With("can't update feature gates for scheduling group %s/%s: %w", sg.Namespace, sg.Name, err)
		}
	}
	for _, sg := range rtx.SchedulerGroups {
		if err := patchFeatures[scope.SchedulerGroup](ctx, t.Client, sg, fs); err != nil {
			return task.Fail().With("can't update feature gates for scheduler group %s/%s: %w", sg.Namespace, sg.Name, err)
		}
	}
	for _, kvg := range rtx.TiKVGroups {
		if err := patchFeatures[scope.TiKVGroup](ctx, t.Client, kvg, fs); err != nil {
			return task.Fail().With("can't update feature gates for tikv group %s/%s: %w", kvg.Namespace, kvg.Name, err)
		}
	}
	for _, fg := range rtx.TiFlashGroups {
		if err := patchFeatures[scope.TiFlashGroup](ctx, t.Client, fg, fs); err != nil {
			return task.Fail().With("can't update feature gates for tiflash group %s/%s: %w", fg.Namespace, fg.Name, err)
		}
	}
	for _, dbg := range rtx.TiDBGroups {
		if err := patchFeatures[scope.TiDBGroup](ctx, t.Client, dbg, fs); err != nil {
			return task.Fail().With("can't update feature gates for tidb group %s/%s: %w", dbg.Namespace, dbg.Name, err)
		}
	}

	for _, cg := range rtx.TiCDCGroups {
		if err := patchFeatures[scope.TiCDCGroup](ctx, t.Client, cg, fs); err != nil {
			return task.Fail().With("can't update feature gates for ticdc group %s/%s: %w", cg.Namespace, cg.Name, err)
		}
	}

	for _, pg := range rtx.TiProxyGroups {
		if err := patchFeatures[scope.TiProxyGroup](ctx, t.Client, pg, fs); err != nil {
			return task.Fail().With("can't update feature gates for tiproxy group %s/%s: %w", pg.Namespace, pg.Name, err)
		}
	}

	return task.Complete().With("features of all groups are updated")
}

type Patch struct {
	Metadata Metadata `json:"metadata"`
	Spec     Spec     `json:"spec"`
}

type Metadata struct {
	ResourceVersion string `json:"resourceVersion"`
}

type Spec struct {
	Features []metav1alpha1.Feature `json:"features"`
}

func patchFeatures[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](ctx context.Context, c client.Client, obj F, fs []metav1alpha1.Feature) error {
	curFs := coreutil.Features[S](obj)
	if reflect.DeepEqual(fs, curFs) {
		return nil
	}

	curSet := sets.New(curFs...)
	updateSet := sets.New(fs...)

	add := updateSet.Difference(curSet)
	del := curSet.Difference(updateSet)

	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("update features", "add", add.UnsortedList(), "del", del.UnsortedList(), "ns", obj.GetNamespace(), "name", obj.GetName())

	p := Patch{
		Metadata: Metadata{
			ResourceVersion: obj.GetResourceVersion(),
		},
		Spec: Spec{
			Features: fs,
		},
	}
	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}

	if err := c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, data)); err != nil {
		return fmt.Errorf("cannot update features for %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	return nil
}
