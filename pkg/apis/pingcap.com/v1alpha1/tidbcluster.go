// Copyright 2018 PingCAP, Inc.
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

func (mt MemberType) String() string {
	return string(mt)
}

func (tc TidbCluster) PDUpgrading() bool {
	return tc.Status.PD.Phase == UpgradePhase
}

func (tc TidbCluster) TiKVUpgrading() bool {
	return tc.Status.TiKV.Phase == UpgradePhase
}
