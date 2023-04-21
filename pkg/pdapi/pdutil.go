// Copyright 2023 PingCAP, Inc.
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

package pdapi

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

// IsClusterStable checks if cluster is stable by quering PD and checking status of all stores
// we check PD directly instead of checking the same information already available in the CR status to get the most up to data
// stores status in PD could be up to 20 seconds stale https://docs.pingcap.com/tidb/stable/tidb-scheduling#information-collection
func IsClusterStable(pdClient PDClient) string {
	storesInfo, err := pdClient.GetStores()
	if err != nil {
		return fmt.Sprintf("can't access PD: %s", err)
	}

	for _, store := range storesInfo.Stores {
		if store.Store == nil {
			return "missing data for one of its stores"
		}
		if store.Store.StateName != v1alpha1.TiKVStateUp && store.Store.StateName != v1alpha1.TiKVStateTombstone {
			return fmt.Sprintf("Strore %d is not up: %s", store.Store.GetId(), store.Store.StateName)
		}
	}

	return ""
}
