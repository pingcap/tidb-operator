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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestEncoder(t *testing.T) {
	cases := []struct {
		desc       string
		topologies []v1alpha1.Topology
		hashes     []string
	}{
		{
			desc: "nil topo will be encoded to empty hash",
			topologies: []v1alpha1.Topology{
				nil,
			},
			hashes: []string{""},
		},
		{
			desc: "normal",
			topologies: []v1alpha1.Topology{
				{
					// 1
					"aaa": "bbb",
				},
				{
					// 2
					"ccc": "ddd",
				},
				{
					"aaa": "bbb",
					"ccc": "ddd",
				},
				{
					"aaa": "bbb",
					"ccc": "ddd",
					// 4
					"eee": "fff",
				},
			},
			hashes: []string{
				string([]byte{1}),
				string([]byte{2}),
				string([]byte{3}),
				string([]byte{7}),
			},
		},
		{
			desc: "more than 8 terms",
			topologies: []v1alpha1.Topology{
				{
					// 1
					"a": "a",
					// 2
					"b": "b",
					// 4
					"c": "c",
					// 8
					"d": "d",
					// 16
					"e": "e",
					// 32
					"f": "f",
					// 64
					"g": "g",
					// 128
					"h": "h",
				},
				{
					// 255 + 1
					"aaa": "aaa",
					// 255 + 2
					"bbb": "bbb",
				},
			},
			hashes: []string{
				string([]byte{255}),
				string([]byte{0, 3}),
			},
		},
		{
			desc: "same topo will be encoded to same hash",
			topologies: []v1alpha1.Topology{
				{
					"aaa": "aaa",
				},
				{
					"bbb": "bbb",
				},
				{
					"aaa": "aaa",
				},
			},
			hashes: []string{
				string([]byte{1}),
				string([]byte{2}),
				string([]byte{1}),
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			e := NewEncoder()
			hashes := []string{}
			for _, topo := range c.topologies {
				hashes = append(hashes, e.Encode(topo))
			}
			assert.Equal(tt, c.hashes, hashes, c.desc)
		})
	}
}

func TestSchedulerAdd(t *testing.T) {
	cases := []struct {
		desc             string
		st               []v1alpha1.ScheduleTopology
		expectedNextAdds []int
		expectedNextDels []int
	}{
		{
			desc: "normal",
			st: []v1alpha1.ScheduleTopology{
				{
					Topology: v1alpha1.Topology{
						"zone": "aaa",
					},
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "bbb",
					},
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "ccc",
					},
				},
			},
			expectedNextAdds: []int{
				0, 1, 2, 0, 1, 2, 0, 1, 2,
			},
			expectedNextDels: []int{
				0, 1, 2, 0, 1, 2, 0, 1, 2,
			},
		},
		{
			desc: "one zone have high weight",
			st: []v1alpha1.ScheduleTopology{
				{
					Topology: v1alpha1.Topology{
						"zone": "aaa",
					},
					Weight: ptr.To[int32](5),
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "bbb",
					},
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "ccc",
					},
				},
			},
			expectedNextAdds: []int{
				0, 1, 2, 0, 0, 0, 0, // 5, 1, 1
				0, 1, 0, 2, 0, 0, 0, // 10, 2, 2
			},
			expectedNextDels: []int{
				0, 1, 0, 2, 0, 0, 0,
				0, 0, 0, 0, 1, 2, 0,
			},
		},
		{
			desc: "one zone have very high weight and no zone is starved",
			st: []v1alpha1.ScheduleTopology{
				{
					Topology: v1alpha1.Topology{
						"zone": "aaa",
					},
					Weight: ptr.To[int32](65535),
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "bbb",
					},
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "ccc",
					},
				},
			},
			expectedNextAdds: []int{
				0, 1, 2, // 1, 1, 1
				0, 0, 0, 0, // ...
			},
			expectedNextDels: []int{
				0, 0, 0, 0,
				1, 2, 0,
			},
		},
		{
			desc: "complex case",
			st: []v1alpha1.ScheduleTopology{
				{
					Topology: v1alpha1.Topology{
						"zone": "aaa",
					},
					Weight: ptr.To[int32](9),
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "bbb",
					},
					Weight: ptr.To[int32](3),
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "ccc",
					},
					Weight: ptr.To[int32](2),
				},
			},
			expectedNextAdds: []int{
				// 1, 1, 1, no starved
				0, 1, 2,
				// 4, 1, 1,
				0, 0, 0,
				// 4, 1+1, 1, 6*9-4*14=-2, 6*3-1*14=4, 6*2-1*14=-2
				1,
				// 4+1, 2, 1, 7*9-4*14=7, 7*3-2*14=-7, 7*2-1*14=0
				0,
				// 5+1, 2, 1, 8*9-5*14=2, 8*3-2*14=-4, 8*2-1*14=2
				0,
				// 6, 2, 1+1, 9*9-6*14=-3, 9*3-2*14=-1, 9*2-1*14=4
				2,
				// 6+1, 2, 2, 10*9-6*14=6, 10*3-2*14=2, 10*2-2*14=-8
				0,
				// 7, 2+1, 2, 11*9-7*14=1, 11*3-2*14=5, 11*2-2*14=-6
				1,
				// 9, 3, 2
				0, 0,
			},
			expectedNextDels: []int{
				// 9-1, 3, 2, 9*14-14*9=0, 3*14-14*3=0, 2*14-14*2=0
				0,
				// 8, 3-1, 2, 8*14-13*9=-5, 3*14-13*3=3, 2*14-13*2=2
				1,
				// 8-1, 2, 2, 8*14-12*9=4, 2*14-12*3=-8, 2*14-12*2=4
				0,
				// 7, 2, 2-1, 7*14-11*9=-1, 2*14-11*3=-5, 2*14-11*2=6
				2,
				// 5, 2, 1
				0, 0,
				// 5, 2-1, 1, 5*14-8*9=-2, 2*14-8*3=4, 1*14-8*2=-2
				1,
				0, 0, 0, 0,
				2, 1, 0,
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			s, err := New(c.st)
			require.NoError(tt, err, c.desc)

			for i, next := range c.expectedNextAdds {
				topo := s.NextAdd()
				assert.Equal(tt, c.st[next].Topology, topo, "index %v", i)
				s.Add(genInstanceName(next, i), topo)
			}
			for i, next := range c.expectedNextDels {
				topoPrefix := strconv.Itoa(next)
				names := s.NextDel()
				for _, name := range names {
					prefix := getInstancePrefix(name)
					assert.Equal(tt, topoPrefix, prefix, "index %v", i)
				}
				fmt.Println(names)
				s.Del(names[0])
			}
		})
	}
}

func genInstanceName(topoIndex, index int) string {
	return strconv.Itoa(topoIndex) + ":" + strconv.Itoa(index)
}

func getInstancePrefix(name string) string {
	ss := strings.Split(name, ":")
	return ss[0]
}
