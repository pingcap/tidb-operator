// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

// BaseTiKVSpec returns the base spec of TiKV servers
func (tg *TiKVGroup) BaseTiKVSpec(tc *TidbCluster) ComponentAccessor {
	return buildTidbClusterComponentAccessor(&tc.Spec, &tg.Spec.ComponentSpec)
}

func (tg *TiKVGroup) TiKVStsDesiredReplicas() int32 {
	// TODO: support failover
	return tg.Spec.Replicas
}

func (tg *TiKVGroup) Scaling() bool {
	return tg.Status.Phase == ScalePhase
}

func (tg *TiKVGroup) Upgrading() bool {
	return tg.Status.Phase == UpgradePhase
}
