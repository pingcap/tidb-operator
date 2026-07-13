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

package waiter

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func WaitForPlacementPolicySynced(
	ctx context.Context,
	c client.Client,
	policy *v1alpha1.PlacementPolicy,
	timeout time.Duration,
) error {
	return WaitForObject(ctx, c, policy, func() error {
		cond := apimeta.FindStatusCondition(policy.Status.Conditions, v1alpha1.CondSynced)
		if cond == nil {
			return fmt.Errorf("placement policy %s/%s's condition %s is not set", policy.Namespace, policy.Name, v1alpha1.CondSynced)
		}
		if cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == policy.Generation {
			return nil
		}

		return fmt.Errorf("placement policy %s/%s's condition %s has unexpected status, expected generation %v, status %v, current status is %v, observed generation: %v, reason: %v, message: %v",
			policy.Namespace,
			policy.Name,
			cond.Type,
			policy.Generation,
			metav1.ConditionTrue,
			cond.Status,
			cond.ObservedGeneration,
			cond.Reason,
			cond.Message,
		)
	}, timeout)
}

func WaitForPlacementRuleExists(
	ctx context.Context,
	pdc pdapi.PDClient,
	policy *v1alpha1.PlacementPolicy,
	expected *pdapi.PlacementRuleGroupBundle,
	timeout time.Duration,
) error {
	if policy == nil {
		return fmt.Errorf("placement policy is nil")
	}
	if expected == nil {
		return fmt.Errorf("expected placement rule bundle is nil")
	}

	groupID := expected.ID
	if groupID == "" && len(expected.Rules) != 0 {
		groupID = expected.Rules[0].GroupID
	}
	if groupID == "" {
		return fmt.Errorf("expected placement rule bundle group ID is empty")
	}
	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		bundle, err := pdc.GetPlacementRuleBundle(ctx, groupID)
		if err != nil {
			lastErr = fmt.Errorf("can't get placement rule bundle %s: %w", groupID, err)
			return false, nil
		}
		if bundle == nil {
			lastErr = fmt.Errorf("placement rule bundle %s is not found", groupID)
			return false, nil
		}
		if err := checkPlacementRulesExist(bundle, expected); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for placement rules in bundle %s timeout: %w", groupID, lastErr)
		}

		return fmt.Errorf("can't wait for placement rules in bundle %s: %w", groupID, err)
	}

	return nil
}

func placementRuleIDs(rules []pdapi.PlacementRule) []string {
	ruleIDs := []string{}
	for _, rule := range rules {
		ruleIDs = append(ruleIDs, rule.ID)
	}
	slices.Sort(ruleIDs)

	return ruleIDs
}

func checkPlacementRulesExist(bundle, expected *pdapi.PlacementRuleGroupBundle) error {
	actualRuleIDs := placementRuleIDs(bundle.Rules)
	expectedRuleIDs := placementRuleIDs(expected.Rules)
	for _, expectedRuleID := range expectedRuleIDs {
		if _, ok := slices.BinarySearch(actualRuleIDs, expectedRuleID); !ok {
			return fmt.Errorf("placement rules are not found in bundle, expected %v, actual %v", expectedRuleIDs, actualRuleIDs)
		}
	}
	return nil
}

func WaitForStoreLabelValue(
	ctx context.Context,
	c client.Client,
	pdc pdapi.PDClient,
	kvg *v1alpha1.TiKVGroup,
	timeout time.Duration,
) error {
	if kvg == nil {
		return fmt.Errorf("TiKVGroup is nil")
	}

	expectedLabels := tikvGroupPlacementLabels(kvg)
	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		storeIDs, pendingTiKVs, err := tikvGroupStoreIDs(ctx, c, kvg)
		if err != nil {
			return false, err
		}
		if len(pendingTiKVs) != 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has TiKV stores without status ID: %v", kvg.Namespace, kvg.Name, pendingTiKVs)
			return false, nil
		}
		if len(storeIDs) == 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has no TiKV stores", kvg.Namespace, kvg.Name)
			return false, nil
		}

		stores, err := pdc.GetStores(ctx)
		if err != nil {
			return false, fmt.Errorf("can't get stores: %w", err)
		}
		if err := checkTiKVGroupStoreLabels(stores, storeIDs, expectedLabels); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for TiKVGroup %s/%s store labels %v timeout: %w", kvg.Namespace, kvg.Name, expectedLabels, lastErr)
		}

		return fmt.Errorf("can't wait for TiKVGroup %s/%s store labels %v: %w", kvg.Namespace, kvg.Name, expectedLabels, err)
	}

	return nil
}

func tikvGroupPlacementLabels(kvg *v1alpha1.TiKVGroup) map[string]string {
	labels := map[string]string{
		v1alpha1.PlacementTiKVGroupLabelKey: coreutil.PlacementTiKVGroupLabelValue(kvg.Namespace, kvg.Name),
	}
	if coreutil.TiKVStorePlacementExclusive(kvg.Spec.Template.Spec.Placement) {
		labels[v1alpha1.PlacementTiKVGroupExclusiveLabelKey] = v1alpha1.PlacementTiKVGroupExclusiveLabelValue
	}

	return labels
}

func checkTiKVGroupStoreLabels(stores *pdapi.StoresInfo, storeIDs map[uint64]string, expectedLabels map[string]string) error {
	missing := []string{}
	mismatched := []string{}
	for storeID, tikvName := range storeIDs {
		store := storeByID(stores, storeID)
		if store == nil {
			missing = append(missing, fmt.Sprintf("%d(%s)", storeID, tikvName))
			continue
		}
		for labelKey, labelValue := range expectedLabels {
			if storeHasLabel(store, labelKey, labelValue) {
				continue
			}
			mismatched = append(mismatched, fmt.Sprintf("%d(%s:%s=%s)", storeID, tikvName, labelKey, labelValue))
		}
	}
	if len(missing) == 0 && len(mismatched) == 0 {
		return nil
	}

	return fmt.Errorf("store labels %v mismatch, missing stores %v, mismatched stores %v: %s", expectedLabels, missing, mismatched, storeSummary(stores))
}

func storeByID(stores *pdapi.StoresInfo, storeID uint64) *pdapi.StoreInfo {
	for _, store := range stores.Stores {
		if store.Store.Id == storeID {
			return store
		}
	}

	return nil
}

func storeHasLabel(store *pdapi.StoreInfo, labelKey, labelValue string) bool {
	for _, label := range store.Store.Labels {
		if label != nil && label.Key == labelKey && label.Value == labelValue {
			return true
		}
	}

	return false
}

func WaitForKeyspaceCreated(ctx context.Context, pdc pdapi.PDClient, name string, timeout time.Duration) (*pdapi.KeyspaceMeta, error) {
	var lastErr error
	var meta *pdapi.KeyspaceMeta
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		var err error
		meta, err = pdc.CreateKeyspace(ctx, name, nil)
		if err != nil {
			lastErr = err
			return false, nil
		}
		if meta == nil {
			lastErr = fmt.Errorf("created keyspace %s has nil metadata", name)
			return false, nil
		}
		if meta.ID == 0 {
			lastErr = fmt.Errorf("created keyspace %s has zero ID", name)
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return nil, fmt.Errorf("wait for keyspace %s created timeout: %w", name, lastErr)
		}

		return nil, fmt.Errorf("can't wait for keyspace %s created: %w", name, err)
	}

	return meta, nil
}

func WaitForStoreRegionCount(
	ctx context.Context,
	c client.Client,
	pdc pdapi.PDClient,
	kvg *v1alpha1.TiKVGroup,
	expected int,
	timeout time.Duration,
) error {
	if kvg == nil {
		return fmt.Errorf("TiKVGroup is nil")
	}

	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		storeIDs, pendingTiKVs, err := tikvGroupStoreIDs(ctx, c, kvg)
		if err != nil {
			return false, err
		}
		if len(pendingTiKVs) != 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has TiKV stores without status ID: %v", kvg.Namespace, kvg.Name, pendingTiKVs)
			return false, nil
		}
		if len(storeIDs) == 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has no TiKV stores", kvg.Namespace, kvg.Name)
			return false, nil
		}

		stores, err := pdc.GetStores(ctx)
		if err != nil {
			return false, fmt.Errorf("can't get stores: %w", err)
		}
		if err := checkTiKVGroupStoreRegionCount(stores, storeIDs, expected); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for TiKVGroup %s/%s store region count %d timeout: %w", kvg.Namespace, kvg.Name, expected, lastErr)
		}

		return fmt.Errorf("can't wait for TiKVGroup %s/%s store region count %d: %w", kvg.Namespace, kvg.Name, expected, err)
	}

	return nil
}

func WaitForStoreRegionCountAtLeast(
	ctx context.Context,
	c client.Client,
	pdc pdapi.PDClient,
	kvg *v1alpha1.TiKVGroup,
	min int,
	timeout time.Duration,
) error {
	if kvg == nil {
		return fmt.Errorf("TiKVGroup is nil")
	}

	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		storeIDs, pendingTiKVs, err := tikvGroupStoreIDs(ctx, c, kvg)
		if err != nil {
			return false, err
		}
		if len(pendingTiKVs) != 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has TiKV stores without status ID: %v", kvg.Namespace, kvg.Name, pendingTiKVs)
			return false, nil
		}
		if len(storeIDs) == 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has no TiKV stores", kvg.Namespace, kvg.Name)
			return false, nil
		}

		stores, err := pdc.GetStores(ctx)
		if err != nil {
			return false, fmt.Errorf("can't get stores: %w", err)
		}
		if err := checkTiKVGroupStoreRegionCountAtLeast(stores, storeIDs, min); err != nil {
			lastErr = err
			return false, nil
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for TiKVGroup %s/%s store region count at least %d timeout: %w", kvg.Namespace, kvg.Name, min, lastErr)
		}

		return fmt.Errorf("can't wait for TiKVGroup %s/%s store region count at least %d: %w", kvg.Namespace, kvg.Name, min, err)
	}

	return nil
}

func checkTiKVGroupStoreRegionCount(stores *pdapi.StoresInfo, storeIDs map[uint64]string, expected int) error {
	missing := []string{}
	noStatus := []string{}
	mismatched := []string{}
	for storeID, tikvName := range storeIDs {
		store := storeByID(stores, storeID)
		if store == nil {
			missing = append(missing, fmt.Sprintf("%d(%s)", storeID, tikvName))
			continue
		}
		if store.Status == nil {
			noStatus = append(noStatus, fmt.Sprintf("%d(%s)", storeID, tikvName))
			continue
		}
		if store.Status.RegionCount != expected {
			mismatched = append(mismatched, fmt.Sprintf("%d(%s):%d", storeID, tikvName, store.Status.RegionCount))
		}
	}
	if len(missing) == 0 && len(noStatus) == 0 && len(mismatched) == 0 {
		return nil
	}

	return fmt.Errorf("store region count mismatch, expected %d, missing stores %v, stores without status %v, mismatched stores %v: %s",
		expected, missing, noStatus, mismatched, storeSummary(stores))
}

func checkTiKVGroupStoreRegionCountAtLeast(stores *pdapi.StoresInfo, storeIDs map[uint64]string, min int) error {
	missing := []string{}
	noStatus := []string{}
	mismatched := []string{}
	for storeID, tikvName := range storeIDs {
		store := storeByID(stores, storeID)
		if store == nil {
			missing = append(missing, fmt.Sprintf("%d(%s)", storeID, tikvName))
			continue
		}
		if store.Status == nil {
			noStatus = append(noStatus, fmt.Sprintf("%d(%s)", storeID, tikvName))
			continue
		}
		if store.Status.RegionCount < min {
			mismatched = append(mismatched, fmt.Sprintf("%d(%s):%d", storeID, tikvName, store.Status.RegionCount))
		}
	}
	if len(missing) == 0 && len(noStatus) == 0 && len(mismatched) == 0 {
		return nil
	}

	return fmt.Errorf("store region count mismatch, expected at least %d, missing stores %v, stores without status %v, mismatched stores %v: %s",
		min, missing, noStatus, mismatched, storeSummary(stores))
}

func WaitForKeyspaceRegionsScheduled(
	ctx context.Context,
	c client.Client,
	pdc pdapi.PDClient,
	keyspaceID string,
	kvg *v1alpha1.TiKVGroup,
	count int,
	timeout time.Duration,
) error {
	if kvg == nil {
		return fmt.Errorf("TiKVGroup is nil")
	}

	var lastErr error
	if err := wait.PollUntilContextTimeout(ctx, Poll, timeout, true, func(ctx context.Context) (bool, error) {
		allowedStoreIDs, pendingTiKVs, err := tikvGroupStoreIDs(ctx, c, kvg)
		if err != nil {
			return false, err
		}
		if len(pendingTiKVs) != 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has TiKV stores without status ID: %v", kvg.Namespace, kvg.Name, pendingTiKVs)
			return false, nil
		}
		if len(allowedStoreIDs) == 0 {
			lastErr = fmt.Errorf("TiKVGroup %s/%s has no TiKV stores with status ID", kvg.Namespace, kvg.Name)
			return false, nil
		}

		regions, err := pdc.ListKeyspaceRegions(ctx, keyspaceID)
		if err != nil {
			return false, fmt.Errorf("can't list keyspace %s regions: %w", keyspaceID, err)
		}
		if len(regions.Regions) == 0 {
			lastErr = fmt.Errorf("keyspace %s has no regions", keyspaceID)
			return false, nil
		}
		if regions.Count != len(regions.Regions) {
			lastErr = fmt.Errorf("keyspace %s regions count mismatch, expected %d, got %d", keyspaceID, len(regions.Regions), regions.Count)
			return false, nil
		}

		for _, region := range regions.Regions {
			if len(region.PendingPeers) != 0 {
				lastErr = fmt.Errorf("region %d has pending peers: %s", region.ID, regionPlacementSummary([]*pdapi.RegionInfo{region}, allowedStoreIDs))
				return false, nil
			}
			if len(region.DownPeers) != 0 {
				lastErr = fmt.Errorf("region %d has down peers: %s", region.ID, regionPlacementSummary([]*pdapi.RegionInfo{region}, allowedStoreIDs))
				return false, nil
			}
			voters := voterPeers(region)
			if len(voters) != count {
				lastErr = fmt.Errorf("region %d voters mismatch: %s", region.ID, regionPlacementSummary([]*pdapi.RegionInfo{region}, allowedStoreIDs))
				return false, nil
			}
			for _, peer := range voters {
				if _, ok := allowedStoreIDs[peer.GetStoreId()]; !ok {
					lastErr = fmt.Errorf("region %d peer store %d is not in TiKVGroup %s/%s stores %s: %s",
						region.ID, peer.GetStoreId(), kvg.Namespace, kvg.Name, storeIDSummary(allowedStoreIDs), regionPlacementSummary([]*pdapi.RegionInfo{region}, allowedStoreIDs))
					return false, nil
				}
			}
		}

		return true, nil
	}); err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("wait for keyspace %s regions scheduled to TiKVGroup %s/%s timeout: %w", keyspaceID, kvg.Namespace, kvg.Name, lastErr)
		}

		return fmt.Errorf("can't wait for keyspace %s regions scheduled to TiKVGroup %s/%s: %w", keyspaceID, kvg.Namespace, kvg.Name, err)
	}

	return nil
}

func tikvGroupStoreIDs(ctx context.Context, c client.Client, kvg *v1alpha1.TiKVGroup) (map[uint64]string, []string, error) {
	tikvs, err := apicall.ListInstances[scope.TiKVGroup](ctx, c, kvg)
	if err != nil {
		return nil, nil, fmt.Errorf("can't list TiKV instances for TiKVGroup %s/%s: %w", kvg.Namespace, kvg.Name, err)
	}

	storeIDs := map[uint64]string{}
	pendingTiKVs := []string{}
	for _, tikv := range tikvs {
		if tikv.Status.ID == "" {
			pendingTiKVs = append(pendingTiKVs, tikv.Name)
			continue
		}
		storeID, err := strconv.ParseUint(tikv.Status.ID, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("TiKV %s/%s has invalid store ID %q: %w", tikv.Namespace, tikv.Name, tikv.Status.ID, err)
		}
		storeIDs[storeID] = tikv.Name
	}
	slices.Sort(pendingTiKVs)

	return storeIDs, pendingTiKVs, nil
}

func storeSummary(stores *pdapi.StoresInfo) string {
	var parts []string
	for _, store := range stores.Stores {
		var labels []string
		for _, label := range store.Store.Labels {
			if label == nil {
				continue
			}
			labels = append(labels, fmt.Sprintf("%s=%s", label.Key, label.Value))
		}
		regionCount := 0
		if store.Status != nil {
			regionCount = store.Status.RegionCount
		}
		parts = append(parts, fmt.Sprintf("store{id:%d labels:%v regions:%d}", store.Store.Id, labels, regionCount))
	}
	return strings.Join(parts, "; ")
}

func voterPeers(region *pdapi.RegionInfo) []*metapb.Peer {
	var voters []*metapb.Peer
	for _, peer := range region.Peers {
		if peer.GetRole() == metapb.PeerRole_Voter {
			voters = append(voters, peer)
		}
	}
	return voters
}

func regionPlacementSummary(regions []*pdapi.RegionInfo, allowedStoreIDs map[uint64]string) string {
	var parts []string
	for _, region := range regions {
		var peers []string
		for _, peer := range region.Peers {
			tikvName, allowed := allowedStoreIDs[peer.GetStoreId()]
			peers = append(peers, fmt.Sprintf("{store:%d role:%s allowed:%t tikv:%s}", peer.GetStoreId(), peer.GetRole().String(), allowed, tikvName))
		}
		parts = append(parts, fmt.Sprintf("region{id:%d start:%q end:%q peers:%v pending:%d down:%d}", region.ID, region.StartKeyHex, region.EndKeyHex, peers, len(region.PendingPeers), len(region.DownPeers)))
	}
	return strings.Join(parts, "; ")
}

func storeIDSummary(storeIDs map[uint64]string) string {
	ids := make([]uint64, 0, len(storeIDs))
	for storeID := range storeIDs {
		ids = append(ids, storeID)
	}
	slices.Sort(ids)

	parts := make([]string, 0, len(ids))
	for _, storeID := range ids {
		parts = append(parts, fmt.Sprintf("%d(%s)", storeID, storeIDs[storeID]))
	}

	return strings.Join(parts, ",")
}
