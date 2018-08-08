// Copyright 2017 PingCAP, Inc.
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

package schedule

import (
	"math/rand"
	"sync"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server/core"
	"github.com/pingcap/pd/server/namespace"
)

type selectedStores struct {
	mu     sync.Mutex
	stores map[uint64]struct{}
}

func newSelectedStores() *selectedStores {
	return &selectedStores{
		stores: make(map[uint64]struct{}),
	}
}

func (s *selectedStores) put(id uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stores[id]; ok {
		return false
	}
	s.stores[id] = struct{}{}
	return true
}

func (s *selectedStores) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stores = make(map[uint64]struct{})
}

func (s *selectedStores) newFilter() Filter {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make(map[uint64]struct{})
	for id := range s.stores {
		cloned[id] = struct{}{}
	}
	return NewExcludedFilter(nil, cloned)
}

// RegionScatterer scatters regions.
type RegionScatterer struct {
	cluster    Cluster
	classifier namespace.Classifier
	filters    []Filter
	selected   *selectedStores
}

// NewRegionScatterer creates a region scatterer.
func NewRegionScatterer(cluster Cluster, classifier namespace.Classifier) *RegionScatterer {
	filters := []Filter{
		NewStateFilter(),
		NewHealthFilter(),
	}

	return &RegionScatterer{
		cluster:    cluster,
		classifier: classifier,
		filters:    filters,
		selected:   newSelectedStores(),
	}
}

// Scatter relocates the region.
func (r *RegionScatterer) Scatter(region *core.RegionInfo) *Operator {
	if r.cluster.IsRegionHot(region.GetId()) {
		return nil
	}

	if len(region.GetPeers()) != r.cluster.GetMaxReplicas() {
		return nil
	}

	return r.scatterRegion(region)
}

func (r *RegionScatterer) scatterRegion(region *core.RegionInfo) *Operator {
	steps := make([]OperatorStep, 0, len(region.GetPeers()))

	stores := r.collectAvailableStores(region)
	var kind OperatorKind
	for _, peer := range region.GetPeers() {
		if len(stores) == 0 {
			// Reset selected stores if we have no available stores.
			r.selected.reset()
			stores = r.collectAvailableStores(region)
		}

		if r.selected.put(peer.GetStoreId()) {
			delete(stores, peer.GetStoreId())
			continue
		}
		newPeer := r.selectPeerToReplace(stores, region, peer)
		if newPeer == nil {
			continue
		}

		// Remove it from stores and mark it as selected.
		delete(stores, newPeer.GetStoreId())
		r.selected.put(newPeer.GetStoreId())

		op := CreateMovePeerOperator("scatter-peer", r.cluster, region, OpAdmin,
			peer.GetStoreId(), newPeer.GetStoreId(), newPeer.GetId())
		steps = append(steps, op.steps...)
		steps = append(steps, TransferLeader{ToStore: newPeer.GetStoreId()})
		kind |= op.Kind()
	}

	if len(steps) == 0 {
		return nil
	}
	return NewOperator("scatter-region", region.GetId(), region.GetRegionEpoch(), kind, steps...)
}

func (r *RegionScatterer) selectPeerToReplace(stores map[uint64]*core.StoreInfo, region *core.RegionInfo, oldPeer *metapb.Peer) *metapb.Peer {
	// scoreGuard guarantees that the distinct score will not decrease.
	regionStores := r.cluster.GetRegionStores(region)
	sourceStore := r.cluster.GetStore(oldPeer.GetStoreId())
	scoreGuard := NewDistinctScoreFilter(r.cluster.GetLocationLabels(), regionStores, sourceStore)

	candidates := make([]*core.StoreInfo, 0, len(stores))
	for _, store := range stores {
		if scoreGuard.FilterTarget(r.cluster, store) {
			continue
		}
		candidates = append(candidates, store)
	}

	if len(candidates) == 0 {
		return nil
	}

	target := candidates[rand.Intn(len(candidates))]
	newPeer, err := r.cluster.AllocPeer(target.GetId())
	if err != nil {
		return nil
	}
	return newPeer
}

func (r *RegionScatterer) collectAvailableStores(region *core.RegionInfo) map[uint64]*core.StoreInfo {
	namespace := r.classifier.GetRegionNamespace(region)
	filters := []Filter{
		r.selected.newFilter(),
		NewExcludedFilter(nil, region.GetStoreIds()),
		NewNamespaceFilter(r.classifier, namespace),
	}
	filters = append(filters, r.filters...)

	stores := r.cluster.GetStores()
	targets := make(map[uint64]*core.StoreInfo, len(stores))
	for _, store := range stores {
		if !FilterTarget(r.cluster, store, filters) {
			targets[store.GetId()] = store
		}
	}
	return targets
}
