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
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/updater"
)

func HeadlessServiceName(groupName string) string {
	return fmt.Sprintf("%s-ticdc-peer", groupName)
}

func NotOwnerPolicy() updater.PreferPolicy[*runtime.TiCDC] {
	return updater.PreferPolicyFunc[*runtime.TiCDC](func(cdcs []*runtime.TiCDC) []*runtime.TiCDC {
		var notOwner []*runtime.TiCDC
		for _, cdc := range cdcs {
			if !cdc.Status.IsOwner {
				notOwner = append(notOwner, cdc)
			}
		}
		return notOwner
	})
}
