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

package tracker

import (
	"github.com/pingcap/tidb-operator/pkg/client"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
	"github.com/pingcap/tidb-operator/pkg/utils/random"
)

type Tracker[
	O client.Object,
	I client.Object,
] interface {
	Track(owner O, items ...I) Allocator
}

type tracker[
	O client.Object,
	I client.Object,
] struct {
	as maputil.Map[string, Allocator]
}

const suffixLen = 6

func WithRandomSuffix(prefix string) NewNameFunc {
	return func() string {
		return prefix + "-" + random.Random(suffixLen)
	}
}

func (t *tracker[O, I]) Track(owner O, items ...I) Allocator {
	key := client.ObjectKeyFromObject(owner)
	a, ok := t.as.Load(key.String())
	if !ok {
		a = NewAllocator(WithRandomSuffix(owner.GetName()))
		t.as.Store(key.String(), a)
	}

	var names []string
	for _, item := range items {
		names = append(names, item.GetName())
	}

	a.Observed(names...)

	return a
}

func New[
	O client.Object,
	I client.Object,
]() Tracker[O, I] {
	return &tracker[O, I]{
		as: maputil.Map[string, Allocator]{},
	}
}

type Allocator interface {
	Allocate(index int) string
	Observed(names ...string)
}

type allocator struct {
	observed []string
	// unseen record all names which are allocated but not observed.
	unseen []string

	// -1: means name is observed
	// >=0: means index of unseen map
	allocated map[string]int

	newNameFunc NewNameFunc
}

func (a *allocator) Allocate(index int) string {
	if index >= len(a.unseen) {
		for range index - len(a.unseen) + 1 {
			a.allocate()
		}
	}

	name := a.unseen[index]
	return name
}

func (a *allocator) allocate() {
	for range 100 {
		name := a.newNameFunc()
		if _, ok := a.allocated[name]; !ok {
			a.unseen = append(a.unseen, name)
			a.allocated[name] = len(a.unseen) - 1

			return
		}
	}

	panic("cannot allocate a new name")
}

func (a *allocator) Observed(names ...string) {
	allocated := make(map[string]int, len(names)+len(a.unseen))
	for _, name := range names {
		allocated[name] = -1
	}
	var unseen []string
	for _, name := range a.unseen {
		if _, ok := allocated[name]; ok {
			continue
		}
		unseen = append(unseen, name)
		allocated[name] = len(unseen) - 1
	}

	a.observed = names
	a.allocated = allocated
	a.unseen = unseen
}

func NewAllocator(f NewNameFunc) Allocator {
	a := &allocator{
		allocated:   make(map[string]int),
		newNameFunc: f,
	}

	return a
}

type NewNameFunc func() string
