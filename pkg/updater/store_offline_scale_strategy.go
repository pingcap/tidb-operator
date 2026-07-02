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
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

type storeOfflineScaleStrategy[R runtime.Instance] struct{}

// NewStoreOfflineScaleStrategy returns an offline scale strategy for PD store components (TiKV/TiFlash).
func NewStoreOfflineScaleStrategy[R runtime.Instance]() OfflineScaleStrategy[R] {
	return storeOfflineScaleStrategy[R]{}
}

func (storeOfflineScaleStrategy[R]) ShouldOffline(obj R, trigger OfflineTrigger) bool {
	if trigger != OfflineOnDelete {
		return false
	}
	return !obj.IsOffline() &&
		!meta.IsStatusConditionTrue(obj.Conditions(), v1alpha1.StoreOfflinedConditionType)
}

func (storeOfflineScaleStrategy[R]) ChooseOfflineToRevive(items []R, _ OfflineScaleContext) (R, bool) {
	if len(items) == 0 {
		var zero R
		return zero, false
	}
	return items[0], true
}

func (storeOfflineScaleStrategy[R]) OfflineRevivePatch(_ R) OfflineRevivePatch {
	return OfflineRevivePatch{ClearOffline: true}
}
