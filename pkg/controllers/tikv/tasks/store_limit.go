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
	"fmt"
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const incomingScaleInLimit = 0.000001

// Original values are saved before PD writes so retries do not overwrite them
// with reduced rates. The record belongs only to this instance's store.
type originalStoreLimits struct {
	AddPeer          float64
	TransferLeaderIn float64
}

func TaskStoreLimitAdjust(state *ReconcileContext, c client.Client, m pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("StoreLimitAdjust", func(ctx context.Context) task.Result {
		kv := state.Object()
		offline := coreutil.IsOffline[scope.TiKV](kv)
		saved := kv.Annotations[v1alpha1.AnnoKeyStoreLimits]
		if offline && !state.FeatureGates().Enabled(meta.TiKVSequentialScaleIn) || !offline && saved == "" {
			return task.Complete().With("no store limit adjustment needed")
		}
		if !state.PDSynced {
			return task.Wait().With("pd is not synced")
		}
		if state.Store == nil || state.GetStoreState() == v1alpha1.StoreStateRemoved {
			return task.Complete().With("store does not exist or is removed")
		}
		pc, ok := state.GetPDClient(m)
		if !ok {
			return task.Wait().With("pd client is not registered")
		}
		if err := adjustStoreLimits(ctx, c, pc.Underlay(), kv, state.GetStoreID(), offline); err != nil {
			return task.Fail().With("cannot adjust incoming store limits: %v", err)
		}
		return task.Complete().With("incoming store limits adjusted")
	})
}

func adjustStoreLimits(ctx context.Context, c client.Client, pd pdapi.PDClient, kv *v1alpha1.TiKV, id string, offline bool) error {
	limits, err := pd.GetStoreLimits(ctx)
	if err != nil {
		return err
	}
	current := limits[id]
	if current.AddPeer == nil || current.TransferLeaderIn == nil {
		return fmt.Errorf("store %s requires add-peer and transfer-leader-in limit support", id)
	}
	var original originalStoreLimits
	saved := kv.Annotations[v1alpha1.AnnoKeyStoreLimits]
	if saved != "" {
		original, err = parseOriginalStoreLimits(saved)
		if err != nil {
			return err
		}
	} else {
		original = originalStoreLimits{AddPeer: *current.AddPeer, TransferLeaderIn: *current.TransferLeaderIn}
		if err := saveOriginalStoreLimits(ctx, c, kv, &original); err != nil {
			return err
		}
	}
	for _, limit := range []struct {
		typ               pdapi.StoreLimitType
		current, original float64
	}{
		{pdapi.StoreLimitAddPeer, *current.AddPeer, original.AddPeer},
		{pdapi.StoreLimitTransferLeaderIn, *current.TransferLeaderIn, original.TransferLeaderIn},
	} {
		target := incomingScaleInLimit
		if !offline {
			// Preserve a value changed by an external writer.
			if limit.current != incomingScaleInLimit {
				continue
			}
			target = limit.original
		}
		if limit.current == target {
			continue
		}
		if err := pd.SetStoreLimit(ctx, id, limit.typ, target); err != nil {
			return err
		}
	}
	if !offline {
		return saveOriginalStoreLimits(ctx, c, kv, nil)
	}
	return nil
}

func saveOriginalStoreLimits(ctx context.Context, c client.Client, kv *v1alpha1.TiKV, limits *originalStoreLimits) error {
	before := kv.DeepCopy()
	if limits == nil {
		delete(kv.Annotations, v1alpha1.AnnoKeyStoreLimits)
	} else {
		if kv.Annotations == nil {
			kv.Annotations = make(map[string]string)
		}
		kv.Annotations[v1alpha1.AnnoKeyStoreLimits] = fmt.Sprintf("add-peer:%g,transfer-leader-in:%g", limits.AddPeer, limits.TransferLeaderIn)
	}
	return c.Patch(ctx, kv, client.MergeFrom(before))
}

func parseOriginalStoreLimits(value string) (originalStoreLimits, error) {
	values := make(map[string]float64)
	for entry := range strings.SplitSeq(value, ",") {
		name, raw, ok := strings.Cut(entry, ":")
		if !ok {
			return originalStoreLimits{}, fmt.Errorf("invalid store limit entry %q", entry)
		}
		rate, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return originalStoreLimits{}, fmt.Errorf("invalid store limit %q: %w", entry, err)
		}
		values[name] = rate
	}
	addPeer, hasAddPeer := values["add-peer"]
	transferLeaderIn, hasTransferLeaderIn := values["transfer-leader-in"]
	if len(values) != 2 || !hasAddPeer || !hasTransferLeaderIn {
		return originalStoreLimits{}, fmt.Errorf("store limits must contain add-peer and transfer-leader-in")
	}
	return originalStoreLimits{AddPeer: addPeer, TransferLeaderIn: transferLeaderIn}, nil
}
