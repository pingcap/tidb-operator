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

func TestAllocator(t *testing.T) {
	cases := []struct {
		desc    string
		current []string
		count   int
		// assume these allocated names are observed
		observedIndex []int
		// unexpected observed names
		additionalObserved []string
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
	}

	for i := range cases {
		c := &cases[i]

		t.Run(c.desc, func(tt *testing.T) {
			a := NewAllocator(WithRandomSuffix("hello"))
			a.Observed(c.current...)
			var previous []string
			for index := range c.count {
				name := a.Allocate(index)
				previous = append(previous, name)
			}

			observed := []string{}
			for _, index := range c.observedIndex {
				// Some observed names are from allocated new names
				if index >= len(c.current) {
					c.count -= 1
					observed = append(observed, previous[index-len(c.current)])
				} else {
					observed = append(observed, c.current[index])
				}
			}
			observed = append(observed, c.additionalObserved...)
			a.Observed(observed...)

			allocated := sets.New(previous...)
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
