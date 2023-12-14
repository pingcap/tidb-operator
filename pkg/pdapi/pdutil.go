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
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	pd "github.com/tikv/pd/client/http"
)

// IsTiKVStable checks if cluster is stable by quering PD and checking status of all stores
// we check PD directly instead of checking the same information already available in the CR status to get the most up to data
// stores status in PD could be up to 20 seconds stale https://docs.pingcap.com/tidb/stable/tidb-scheduling#information-collection
func IsTiKVStable(pdClient pd.Client) string {
	storesInfo, err := pdClient.GetStores(context.TODO())
	if err != nil {
		return fmt.Sprintf("can't access PD: %s", err)
	}

	for _, store := range storesInfo.Stores {
		if store.Store.StateName != v1alpha1.TiKVStateUp && store.Store.StateName != v1alpha1.TiKVStateTombstone {
			return fmt.Sprintf("Strore %d is not up: %s", store.Store.ID, store.Store.StateName)
		}
	}

	return ""
}

// IsPDStable queries PD to verify that there are quorum + 1 healthy PDs
func IsPDStable(pdClient pd.Client) string {
	healthInfo, err := pdClient.GetHealth(context.TODO())
	if err != nil {
		return fmt.Sprintf("can't access PD: %s", err)
	}

	total := 0
	healthy := 0
	for _, memberHealth := range healthInfo.Healths {
		total++
		if memberHealth.Health {
			healthy++
		}
	}

	if healthy > total/2+1 {
		return ""
	} else {
		return fmt.Sprintf("Only %d out of %d PDs are healthy", healthy, total)
	}
}

func BeginEvictLeader(ctx context.Context, pdClient pd.Client, storeID uint64) error {
	err := pdClient.CreateScheduler(ctx, "evict-leader-scheduler", storeID)
	if err != nil {
		return err
	}

	// pd will return an error with the body contains "scheduler existed" if the scheduler already exists
	// this is not the standard response.
	// so these lines are just a workaround for now:
	//   - make a new request to get all schedulers
	//   - return nil if the scheduler already exists
	//
	// when PD returns standard json response, we should get rid of this verbose code.
	//evictLeaderSchedulers, err := pdClient.GetEvictLeaderSchedulers()
	evictLeaderSchedulers, err := GetEvictLeaderSchedulers(ctx, pdClient)
	if err != nil {
		return err
	}
	for _, scheduler := range evictLeaderSchedulers {
		if scheduler == getLeaderEvictSchedulerStr(storeID) {
			return nil
		}
	}
	return errors.New("scheduler not existed")
}

func EndEvictLeader(ctx context.Context, pdClient pd.Client, storeID uint64) error {
	sName := getLeaderEvictSchedulerStr(storeID)
	// delete
	err := pdClient.DeleteScheduler(ctx, "evict-leader-scheduler", storeID)
	if err != nil {
		return err
	}

	// pd will return an error with the body contains "scheduler not found" if the scheduler is not found
	// this is not the standard response.
	// so these lines are just a workaround for now:
	//   - make a new request to get all schedulers
	//   - return nil if the scheduler is not found
	//
	// when PD returns standard json response, we should get rid of this verbose code.
	evictLeaderSchedulers, err := GetEvictLeaderSchedulers(ctx, pdClient)
	if err != nil {
		return err
	}
	for _, s := range evictLeaderSchedulers {
		if s == sName {
			return fmt.Errorf("end leader evict scheduler failed,the store:[%d]'s leader evict scheduler is still exist", storeID)
		}
	}

	return nil
}

func getLeaderEvictSchedulerStr(storeID uint64) string {
	return fmt.Sprintf("%s-%d", "evict-leader-scheduler", storeID)
}

func GetEvictLeaderSchedulers(ctx context.Context, pdClient pd.Client) ([]string, error) {
	schedulers, err := pdClient.GetSchedulers(ctx)
	if err != nil {
		return nil, err
	}

	var evicts []string
	for _, scheduler := range schedulers {
		if strings.HasPrefix(scheduler, evictSchedulerLeader) {
			evicts = append(evicts, scheduler)
		}
	}

	evictSchedulers, err := filterLeaderEvictScheduler(pdClient, evicts)
	if err != nil {
		return nil, err
	}
	return evictSchedulers, nil
}

func filterLeaderEvictScheduler(pdClient pd.Client, evictLeaderSchedulers []string) ([]string, error) {
	var schedulerIds []string
	if len(evictLeaderSchedulers) == 1 && evictLeaderSchedulers[0] == evictSchedulerLeader {
		// If there is only one evcit scehduler entry without store ID postfix.
		// We should get the store IDs via scheduler config API and append them
		// to provide consistent results.
		c, err := pdClient.GetEvictLeaderSchedulerConfig(context.TODO())
		if err != nil {
			return nil, err
		}
		for k := range c.StoreIDWithRanges {
			schedulerIds = append(schedulerIds, fmt.Sprintf("%s-%v", evictSchedulerLeader, k))
		}
	} else {
		schedulerIds = append(schedulerIds, evictLeaderSchedulers...)
	}
	return schedulerIds, nil
}

func GetEvictLeaderSchedulersForStores(ctx context.Context, pdClient pd.Client, storeIDs ...uint64) (map[uint64]string, error) {
	schedulers, err := pdClient.GetSchedulers(ctx)
	if err != nil {
		return nil, err
	}

	find := func(id uint64) string {
		for _, scheduler := range schedulers {
			sName := getLeaderEvictSchedulerStr(id)
			if scheduler == sName {
				return scheduler
			}
		}
		return ""
	}

	result := make(map[uint64]string)
	for _, id := range storeIDs {
		if scheduler := find(id); scheduler != "" {
			result[id] = scheduler
		}
	}

	return result, nil
}
