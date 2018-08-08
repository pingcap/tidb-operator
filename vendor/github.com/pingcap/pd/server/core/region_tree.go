// Copyright 2016 PingCAP, Inc.
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

package core

import (
	"bytes"

	"github.com/google/btree"
	"github.com/pingcap/kvproto/pkg/metapb"
	log "github.com/sirupsen/logrus"
)

var _ btree.Item = &regionItem{}

type regionItem struct {
	region *metapb.Region
}

// Less returns true if the region start key is less than the other.
func (r *regionItem) Less(other btree.Item) bool {
	left := r.region.GetStartKey()
	right := other.(*regionItem).region.GetStartKey()
	return bytes.Compare(left, right) < 0
}

func (r *regionItem) Contains(key []byte) bool {
	start, end := r.region.GetStartKey(), r.region.GetEndKey()
	return bytes.Compare(key, start) >= 0 && (len(end) == 0 || bytes.Compare(key, end) < 0)
}

const (
	defaultBTreeDegree = 64
)

type regionTree struct {
	tree *btree.BTree
}

func newRegionTree() *regionTree {
	return &regionTree{
		tree: btree.New(defaultBTreeDegree),
	}
}

func (t *regionTree) length() int {
	return t.tree.Len()
}

// update updates the tree with the region.
// It finds and deletes all the overlapped regions first, and then
// insert the region.
func (t *regionTree) update(region *metapb.Region) []*metapb.Region {
	item := &regionItem{region: region}

	// note that find() gets the last item that is less or equal than the region.
	// in the case: |_______a_______|_____b_____|___c___|
	// new region is     |______d______|
	// find() will return regionItem of region_a
	// and both startKey of region_a and region_b are less than endKey of region_d,
	// thus they are regarded as overlapped regions.
	result := t.find(region)
	if result == nil {
		result = item
	}

	var overlaps []*metapb.Region
	t.tree.AscendGreaterOrEqual(result, func(i btree.Item) bool {
		over := i.(*regionItem)
		if len(region.EndKey) > 0 && bytes.Compare(region.EndKey, over.region.StartKey) <= 0 {
			return false
		}
		overlaps = append(overlaps, over.region)
		return true
	})

	for _, item := range overlaps {
		log.Debugf("[region %d] delete region {%v}, cause overlapping with region {%v}", item.GetId(), item, region)
		t.tree.Delete(&regionItem{item})
	}

	t.tree.ReplaceOrInsert(item)

	return overlaps
}

// remove removes a region if the region is in the tree.
// It will do nothing if it cannot find the region or the found region
// is not the same with the region.
func (t *regionTree) remove(region *metapb.Region) {
	result := t.find(region)
	if result == nil || result.region.GetId() != region.GetId() {
		return
	}

	t.tree.Delete(result)
}

// search returns a region that contains the key.
func (t *regionTree) search(regionKey []byte) *metapb.Region {
	region := &metapb.Region{StartKey: regionKey}
	result := t.find(region)
	if result == nil {
		return nil
	}
	return result.region
}

// find is a helper function to find an item that contains the regions start
// key.
func (t *regionTree) find(region *metapb.Region) *regionItem {
	item := &regionItem{region: region}

	var result *regionItem
	t.tree.DescendLessOrEqual(item, func(i btree.Item) bool {
		result = i.(*regionItem)
		return false
	})

	if result == nil || !result.Contains(region.StartKey) {
		return nil
	}

	return result
}

func (t *regionTree) scanRange(startKey []byte, f func(*metapb.Region) bool) {
	startItem := &regionItem{region: &metapb.Region{StartKey: startKey}}
	t.tree.AscendGreaterOrEqual(startItem, func(item btree.Item) bool {
		return f(item.(*regionItem).region)
	})
}

func (t *regionTree) getAdjacentRegions(region *metapb.Region) (*regionItem, *regionItem) {
	item := &regionItem{region: &metapb.Region{StartKey: region.StartKey}}
	var prev, next *regionItem
	t.tree.AscendGreaterOrEqual(item, func(i btree.Item) bool {
		if bytes.Compare(item.region.StartKey, i.(*regionItem).region.StartKey) == 0 {
			return true
		}
		next = i.(*regionItem)
		return false
	})
	t.tree.DescendLessOrEqual(item, func(i btree.Item) bool {
		if bytes.Compare(item.region.StartKey, i.(*regionItem).region.StartKey) == 0 {
			return true
		}
		prev = i.(*regionItem)
		return false
	})
	return prev, next
}
