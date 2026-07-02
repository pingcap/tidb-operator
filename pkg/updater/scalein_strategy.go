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

package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

// OfflineTrigger indicates which scale-in path is requesting an offline decision.
type OfflineTrigger int

const (
	// OfflineOnScaleInUpdate is used when scaling in from the update pool.
	// Callers should gate rolling-update cases (for example outdated.Len() == 0) before calling.
	OfflineOnScaleInUpdate OfflineTrigger = iota
	// OfflineOnDelete is used when deleteInstance would otherwise delete the CR immediately.
	OfflineOnDelete
)

// ScaleInContext carries runtime state needed by ScaleInStrategy.
type ScaleInContext struct {
	Now time.Time
}

// ScaleInRevivePatch describes metadata/spec changes applied when canceling offline.
// When ClearOffline is true, cancelOneOfflining patches the instance (if still offline)
// and returns it to the update pool without recreating via updateOutdated.
type ScaleInRevivePatch struct {
	ClearOffline bool
	Annotations  map[string]*string
}

// ScaleInStrategy customizes scale-in offline and scale-out revival behavior.
type ScaleInStrategy[R runtime.Instance] interface {
	// ShouldOffline reports whether to set spec.offline instead of deleting the CR now.
	ShouldOffline(obj R, trigger OfflineTrigger) bool
	ChooseOfflineToRevive(items []R, ctx ScaleInContext) (chosen R, ok bool)
	OfflineRevivePatch(obj R) ScaleInRevivePatch
}

type noopScaleInStrategy[R runtime.Instance] struct{}

// DefaultScaleInStrategy returns the default scale-in strategy used by most components.
func DefaultScaleInStrategy[R runtime.Instance]() ScaleInStrategy[R] {
	return noopScaleInStrategy[R]{}
}

func (noopScaleInStrategy[R]) ShouldOffline(_ R, _ OfflineTrigger) bool {
	return false
}

func (noopScaleInStrategy[R]) ChooseOfflineToRevive(items []R, _ ScaleInContext) (R, bool) {
	if len(items) == 0 {
		var zero R
		return zero, false
	}
	return items[0], true
}

func (noopScaleInStrategy[R]) OfflineRevivePatch(_ R) ScaleInRevivePatch {
	return ScaleInRevivePatch{}
}

// Apply writes the revive patch to the given object.
func (p ScaleInRevivePatch) Apply(ctx context.Context, c client.Client, obj client.Object) error {
	if !p.ClearOffline && len(p.Annotations) == 0 {
		return nil
	}

	patch := Patch{
		Metadata: Metadata{
			ResourceVersion: obj.GetResourceVersion(),
			Annotations:     p.Annotations,
		},
	}
	if p.ClearOffline {
		patch.Spec = &Spec{
			Offline: false,
		}
	}

	data, err := json.Marshal(&patch)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}

	return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, data))
}
