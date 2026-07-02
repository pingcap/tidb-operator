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

package scalein

import (
	"sort"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/updater"
)

type gracefulScaleInStrategy[R runtime.Instance] struct{}

// NewGracefulScaleInStrategy returns a scale-in strategy for TiProxy graceful scale-in.
func NewGracefulScaleInStrategy[R runtime.Instance]() updater.ScaleInStrategy[R] {
	return gracefulScaleInStrategy[R]{}
}

func (gracefulScaleInStrategy[R]) ShouldOfflineInsteadOfDelete(obj R) bool {
	return needsGracefulOfflineScaleIn(obj) && !obj.IsOffline()
}

func (gracefulScaleInStrategy[R]) ChooseOfflineToRevive(items []R, ctx updater.ScaleInContext) (R, bool) {
	return chooseBeingOfflineToRevive(items, ctx.Now)
}

func (gracefulScaleInStrategy[R]) OfflineRevivePatch(_ R) updater.ScaleInRevivePatch {
	return updater.ScaleInRevivePatch{
		ClearOffline: true,
		Annotations: map[string]*string{
			v1alpha1.AnnoKeyTiProxyGracefulShutdownConnectionsDrained: nil,
		},
	}
}

func needsGracefulOfflineScaleIn(obj runtime.Instance) bool {
	if obj.IsStore() {
		return false
	}
	raw := obj.GetAnnotations()[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds]
	if raw == "" {
		return false
	}
	seconds, err := strconv.ParseInt(raw, 10, 32)
	return err == nil && seconds > 0
}

func revivableForScaleOut(obj runtime.Instance, now time.Time) bool {
	if !needsGracefulOfflineScaleIn(obj) {
		return true
	}
	revivable, err := coreutil.RevivableForGracefulScaleOutFromSources(now, obj.GetAnnotations())
	if err != nil {
		return false
	}
	return revivable
}

func chooseBeingOfflineToRevive[R runtime.Instance](items []R, now time.Time) (R, bool) {
	var revivable []R
	for _, item := range items {
		if revivableForScaleOut(item, now) {
			revivable = append(revivable, item)
		}
	}
	if len(revivable) == 0 {
		var zero R
		return zero, false
	}
	if len(revivable) == 1 {
		return revivable[0], true
	}

	sorted := append([]R(nil), revivable...)
	sort.Slice(sorted, func(i, j int) bool {
		return gracefulShutdownBeginTimeForRevive(sorted[i]).After(gracefulShutdownBeginTimeForRevive(sorted[j]))
	})
	return sorted[0], true
}

func gracefulShutdownBeginTimeForRevive(obj runtime.Instance) time.Time {
	startAt := coreutil.GracefulShutdownBeginTimeFromSources(obj.GetAnnotations())
	if !startAt.IsZero() {
		return startAt
	}
	return time.Now()
}
