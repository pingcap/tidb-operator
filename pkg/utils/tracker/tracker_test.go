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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestAllocate(t *testing.T) {
	cases := []struct {
		desc    string
		current []string
		count   int
		// assume these allocated names are observed
		observedIndex []int
		// unexpected observed names
		additionalObserved []string

		// first array is index tracked before AllocateFactory.New
		// second array is index tracked after AllocateFactory.New but before allcating
		trackedIndex [2][]int
		// if true, names in trackedIndex are taken by other groups
		isTaken bool
	}{
		{
			desc:  "no current, no new observed",
			count: 3,
		},
		{
			desc:          "no current, observe 1 new item",
			count:         3,
			observedIndex: []int{0},
		},
		{
			desc:          "no current, observe unordered items",
			count:         5,
			observedIndex: []int{3, 2},
		},
		{
			desc:               "no current, observe unordered items and unexpected items",
			count:              5,
			observedIndex:      []int{3, 2},
			additionalObserved: []string{"test", "xxx"},
		},
		{
			desc:               "has current and current is observed",
			current:            []string{"aaa", "bbb"},
			count:              10,
			observedIndex:      []int{0, 1, 5},
			additionalObserved: []string{"test", "xxx"},
		},
		{
			desc:               "has current and some items are deleted",
			current:            []string{"aaa", "bbb"},
			count:              10,
			observedIndex:      []int{1, 5},
			additionalObserved: []string{"test", "xxx"},
		},
		{
			desc:               "only unexpected items",
			count:              10,
			additionalObserved: []string{"test", "xxx"},
		},
		{
			desc:               "some items are tracked",
			count:              10,
			additionalObserved: []string{"test", "xxx"},

			trackedIndex: [2][]int{
				{0, 1, 2},
				{3, 4, 5},
			},
		},
		{
			desc:               "some items are taken by others",
			count:              10,
			additionalObserved: []string{"test", "xxx"},

			trackedIndex: [2][]int{
				{0, 1, 2},
				{3, 4, 5},
			},
			isTaken: true,
		},
	}

	for i := range cases {
		c := &cases[i]

		t.Run(c.desc, func(tt *testing.T) {
			f := New()
			af := f.AllocateFactory("xxx")
			t := f.Tracker("xxx")

			var all []string
			for _, name := range c.current {
				t.Track("xxx", name, "hello")
				all = append(all, name)
			}

			a := af.New("xxx", "hello", c.current...)
			var previous []string
			for index := range c.count {
				name := a.Allocate(index)
				previous = append(previous, name)
				all = append(all, name)
			}
			allocated := sets.New(previous...)

			observed := []string{}
			for _, index := range c.observedIndex {
				// Some observed names are from allocated new names
				if index >= len(c.current) {
					c.count -= 1
				}
				observed = append(observed, all[index])
			}
			observed = append(observed, c.additionalObserved...)

			for _, index := range c.trackedIndex[0] {
				group := "hello"
				if c.isTaken {
					group = "othergroup"
					// if the instance has been taken by others,
					// delete it from the allocated names
					allocated.Delete(all[index])
					c.count--
				}
				t.Track("xxx", all[index], group)
			}

			a = af.New("xxx", "hello", observed...)

			for _, index := range c.trackedIndex[1] {
				group := "hello"
				if c.isTaken {
					group = "othergroup"
					// track will not affect allocator if it's created.
				}
				t.Track("xxx", all[index], group)
			}

			// Try to allocate all previous allocated names
			for index := range c.count {
				name := a.Allocate(index)
				assert.True(tt, allocated.Has(name), "%v %s should be in previous set", index, name)
			}

			name := a.Allocate(c.count)
			// Allocated names are used up, the new name should not be in previous set
			assert.False(tt, allocated.Has(name), "allocate a new name %s if previous allocated names are used up", name)
		})
	}
}
