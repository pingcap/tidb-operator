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

package topology

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

// This interface defines a scheduler to choose the next topology
type Scheduler interface {
	// Add adds an scheduled instance.
	// All scheduled instances should be added before calling Next()
	Add(name string, topo v1alpha1.Topology)
	// NextAdd returns the topology of the next pending instance
	NextAdd() v1alpha1.Topology
	// Del removes an scheduled instance.
	Del(name string)
	// NextDel returns names which can be choosed to del
	NextDel() []string
}

func New(st []v1alpha1.ScheduleTopology) (Scheduler, error) {
	s := &topologyScheduler{
		e:           NewEncoder(),
		hashToIndex: map[string]int{},
		nameToIndex: map[string]int{},
	}

	for index, topo := range st {
		hash := s.e.Encode(topo.Topology)
		if hash == "" {
			return nil, fmt.Errorf("topology of %vth is empty", index)
		}

		info := topoInfo{
			topo:  topo.Topology,
			names: sets.New[string](),
		}
		if topo.Weight == nil {
			info.weight = 1
		} else {
			info.weight = *topo.Weight
		}

		s.hashToIndex[hash] = index
		s.info = append(s.info, info)
		s.totalWeight += int64(info.weight)
	}

	// append an unknown topo
	s.info = append(s.info, topoInfo{
		names: sets.New[string](),
	})

	return s, nil
}

type topoInfo struct {
	topo   v1alpha1.Topology
	names  sets.Set[string]
	weight int32
}

type topologyScheduler struct {
	e    Encoder
	info []topoInfo
	// topo hash to topo index
	hashToIndex map[string]int
	// instance name to topo index
	nameToIndex map[string]int

	totalWeight int64
	totalCount  int
}

func (s *topologyScheduler) Add(name string, t v1alpha1.Topology) {
	hash := s.e.Encode(t)
	index, ok := s.hashToIndex[hash]
	if !ok {
		// add into unknown topo
		index = len(s.info) - 1
	} else {
		// only count instances not in unknown topo
		s.totalCount += 1
	}

	s.info[index].names.Insert(name)
	s.nameToIndex[name] = index
}

func (s *topologyScheduler) Del(name string) {
	index, ok := s.nameToIndex[name]
	if !ok {
		// cannot found this name, just return
		return
	}

	delete(s.nameToIndex, name)
	s.info[index].names.Delete(name)
	if index != len(s.info)-1 {
		// not unknown topo
		s.totalCount -= 1
	}
}

func (s *topologyScheduler) NextAdd() v1alpha1.Topology {
	// only unknown topo
	if len(s.info) == 1 {
		return nil
	}
	maximum := int64(0)
	choosed := 0
	for i, v := range s.info[:len(s.info)-1] {
		count := v.names.Len()
		// avoid some topos are starved
		if v.names.Len() == 0 {
			// If weight of a topo is very high, countPerWeight*weight ~= totalCount,
			// so add len(topologies) score for empty topo.
			// len(topologies) will always be bigger than the totalCount at the first round.
			// All topos will be scheduled at least one instance at the first round.
			count = -len(s.info) + 1
		}
		score := int64(v.weight)*int64(s.totalCount) - int64(count)*s.totalWeight
		if score > maximum {
			maximum = score
			choosed = i
		}
	}

	return s.info[choosed].topo
}

func (s *topologyScheduler) NextDel() []string {
	unknown := s.info[len(s.info)-1]
	if unknown.names.Len() != 0 {
		return unknown.names.UnsortedList()
	}

	maximum := -int64(s.totalCount+len(s.info)-1) * s.totalWeight
	choosed := 0
	for i, v := range s.info[:len(s.info)-1] {
		count := v.names.Len()
		if count == 0 {
			continue
		}
		if v.names.Len() == 1 {
			count = -len(s.info) + 1
		}
		score := int64(count)*s.totalWeight - int64(s.totalCount)*int64(v.weight)
		if score > maximum {
			maximum = score
			choosed = i
		}
	}

	return s.info[choosed].names.UnsortedList()
}

// Encoder is defined to encode a map to a unique string
// This encoder is designed to avoid iter all topologies in policy multiple times
// Assume there are M toplogies and N instances, time complexity will be optimized from O(M*N) to O(M+N)
type Encoder interface {
	Encode(topo v1alpha1.Topology) string
}

func NewEncoder() Encoder {
	return &encoder{
		kv: map[string]int{},
	}
}

type encoder struct {
	kv       map[string]int
	maxIndex int
}

func (e *encoder) Encode(t v1alpha1.Topology) string {
	var hash []byte
	for k, v := range t {
		count, ok := e.kv[k+":"+v]
		if !ok {
			count = e.maxIndex
			e.maxIndex += 1
			e.kv[k+":"+v] = count
		}
		a := count / 8
		b := count % 8
		if a >= len(hash) {
			for i := 0; i < a-len(hash); i++ {
				hash = append(hash, 0)
			}
			hash = append(hash, 1<<b)
		} else {
			hash[a] |= 1 << b
		}
	}

	return string(hash)
}
