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
	"time"

	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
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
	// ShouldOfflineInsteadOfDelete reports whether ScaleInUpdate should mark the instance
	// offline instead of deleting it. Callers should only consult this on the scale-in-update
	// path and gate rolling-update cases (for example outdated.Len() == 0) before calling.
	ShouldOfflineInsteadOfDelete(obj R) bool
	ChooseOfflineToRevive(items []R, ctx ScaleInContext) (chosen R, ok bool)
	OfflineRevivePatch(obj R) ScaleInRevivePatch
}

type noopScaleInStrategy[R runtime.Instance] struct{}

// DefaultScaleInStrategy returns the default scale-in strategy used by most components.
func DefaultScaleInStrategy[R runtime.Instance]() ScaleInStrategy[R] {
	return noopScaleInStrategy[R]{}
}

func (noopScaleInStrategy[R]) ShouldOfflineInsteadOfDelete(_ R) bool {
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
