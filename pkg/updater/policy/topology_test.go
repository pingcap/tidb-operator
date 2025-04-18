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

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestTopologyPolicy(t *testing.T) {
	const newRev = "new"
	const oldRev = "old"
	cases := []struct {
		desc     string
		existing []*v1alpha1.PD
		add      *v1alpha1.PD
		update   *v1alpha1.PD
		del      string

		expectedAdd      *v1alpha1.PD
		expectedUpdate   *v1alpha1.PD
		expectedPrefered []*v1alpha1.PD
	}{
		{
			desc:        "no existsing, add a new instance, will set topology",
			add:         fakePD("add", newRev, ""),
			expectedAdd: fakePD("add", newRev, "us-west-1a"),
		},

		{
			desc: "existsing updated a, add a new will be scheduled to b",
			existing: []*v1alpha1.PD{
				fakePD("0", newRev, "us-west-1a"),
			},
			add:         fakePD("add", newRev, ""),
			expectedAdd: fakePD("add", newRev, "us-west-1b"),
		},

		{
			desc: "existsing updated ab, add a new will be scheduled to c",
			existing: []*v1alpha1.PD{
				fakePD("0", newRev, "us-west-1a"),
				fakePD("1", newRev, "us-west-1b"),
			},
			add:         fakePD("add", newRev, ""),
			expectedAdd: fakePD("add", newRev, "us-west-1c"),
		},

		{
			desc: "existsing updated abc, add a new will be scheduled to a",
			existing: []*v1alpha1.PD{
				fakePD("0", newRev, "us-west-1a"),
				fakePD("1", newRev, "us-west-1b"),
				fakePD("2", newRev, "us-west-1c"),
			},
			add:         fakePD("add", newRev, ""),
			expectedAdd: fakePD("add", newRev, "us-west-1a"),
		},

		{
			// two new instance will be scheduled to a different zone
			desc: "existsing updated a and old bc, add a new will be scheduled to b",
			existing: []*v1alpha1.PD{
				fakePD("0", newRev, "us-west-1a"),
				fakePD("1", oldRev, "us-west-1b"),
				fakePD("2", oldRev, "us-west-1c"),
			},
			add:         fakePD("add", newRev, ""),
			expectedAdd: fakePD("add", newRev, "us-west-1b"),

			expectedPrefered: []*v1alpha1.PD{
				fakePD("1", oldRev, "us-west-1b"),
			},
		},

		{
			desc: "existsing updated ab and old c, add a new will be scheduled to c",
			existing: []*v1alpha1.PD{
				fakePD("0", newRev, "us-west-1a"),
				fakePD("1", newRev, "us-west-1b"),
				fakePD("2", oldRev, "us-west-1c"),
			},
			add:         fakePD("add", newRev, ""),
			expectedAdd: fakePD("add", newRev, "us-west-1c"),

			expectedPrefered: []*v1alpha1.PD{
				fakePD("2", oldRev, "us-west-1c"),
			},
		},

		{
			desc: "update will keep topology",
			existing: []*v1alpha1.PD{
				fakePD("0", newRev, "us-west-1a"),
				fakePD("1", oldRev, "us-west-1b"),
				fakePD("2", oldRev, "us-west-1c"),
			},
			update:         fakePD("2", newRev, ""),
			expectedUpdate: fakePD("2", newRev, "us-west-1c"),

			expectedPrefered: []*v1alpha1.PD{
				fakePD("1", oldRev, "us-west-1b"),
			},
		},

		{
			desc: "del an existing obj",
			existing: []*v1alpha1.PD{
				fakePD("0", newRev, "us-west-1a"),
				fakePD("1", oldRev, "us-west-1b"),
				fakePD("2", oldRev, "us-west-1c"),
			},
			del: "0",

			expectedPrefered: []*v1alpha1.PD{
				fakePD("1", oldRev, "us-west-1b"),
				fakePD("2", oldRev, "us-west-1c"),
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			policy, err := NewTopologyPolicy([]v1alpha1.ScheduleTopology{
				{
					Topology: v1alpha1.Topology{
						"zone": "us-west-1a",
					},
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "us-west-1b",
					},
				},
				{
					Topology: v1alpha1.Topology{
						"zone": "us-west-1c",
					},
				},
			}, newRev, runtime.FromPDSlice(c.existing)...)
			require.NoError(tt, err)

			instances := map[string]*v1alpha1.PD{}
			for _, in := range c.existing {
				instances[in.GetName()] = in
			}

			if c.add != nil {
				actualAdd := policy.Add(runtime.FromPD(c.add))
				require.Equal(t, c.add, runtime.ToPD(actualAdd)) // in-place update
				assert.Equal(t, c.expectedAdd, c.add)
				instances[c.add.GetName()] = c.add
			}

			if c.update != nil {
				actualUpdate := policy.Update(runtime.FromPD(c.update), runtime.FromPD(instances[c.update.GetName()]))
				require.Equal(t, c.update, runtime.ToPD(actualUpdate)) // in-place update
				assert.Equal(t, c.expectedUpdate, c.update)
				instances[c.update.GetName()] = c.update
			}

			if c.del != "" {
				policy.Delete(c.del)
				delete(instances, c.del)
			}

			var outdated []*v1alpha1.PD
			for _, in := range instances {
				rev, ok := in.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
				if !ok || rev != newRev {
					outdated = append(outdated, in)
				}
			}

			preferred := policy.Prefer(runtime.FromPDSlice(outdated))
			assert.Equal(tt, c.expectedPrefered, runtime.ToPDSlice(preferred))
		})
	}
}

func fakePD(name string, rev string, zone string) *v1alpha1.PD {
	pd := fake.FakeObj(name, fake.Label[v1alpha1.PD](v1alpha1.LabelKeyInstanceRevisionHash, rev))
	if zone != "" {
		pd.Spec.Topology = v1alpha1.Topology{
			"zone": zone,
		}
	}
	return pd
}
