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

package v1alpha1

import "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"

func (t *TiBRGCSpec) GetT2Strategy() *TieredStorageStrategy {
	return t.getStrategy(TieredStorageStrategyNameToT2Storage)
}

func (t *TiBRGCSpec) GetT3Strategy() *TieredStorageStrategy {
	return t.getStrategy(TieredStorageStrategyNameToT3Storage)
}

func (t *TiBRGCSpec) getStrategy(strategyName TieredStorageStrategyName) *TieredStorageStrategy {
	for _, strategy := range t.GCStrategy.TieredStrategies {
		if strategy.Name == strategyName {
			return &strategy
		}
	}
	return nil
}

func (t *TiBRGCSpec) GetPVCOverlay(strategyName TieredStorageStrategyName, pvcName string) *v1alpha1.PersistentVolumeClaimOverlay {
	overlay := t.GetOverlay(strategyName)
	if overlay == nil || overlay.PersistentVolumeClaims == nil {
		return nil
	}
	for _, pvcOverlay := range overlay.PersistentVolumeClaims {
		if pvcOverlay.Name == pvcName {
			return &pvcOverlay.PersistentVolumeClaim
		}
	}
	return nil
}

func (t *TiBRGCSpec) GetOverlay(strategyName TieredStorageStrategyName) *v1alpha1.Overlay {
	for _, pod := range t.Overlay.Pods {
		if pod.Name == strategyName {
			return pod.Overlay
		}
	}
	return nil
}
