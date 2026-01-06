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

package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestSelector(t *testing.T) {
	cases := []struct {
		desc     string
		ps       []PreferPolicy[*runtime.PD]
		allowed  []*runtime.PD
		expected string
	}{
		{
			desc: "no policy",
			allowed: []*runtime.PD{
				fakePD("aaa", true, true),
				fakePD("bbb", true, false),
				fakePD("ccc", true, true),
				fakePD("ddd", true, false),
			},
			expected: "aaa",
		},
		{
			desc: "prefer unready",
			ps: []PreferPolicy[*runtime.PD]{
				PreferUnready[*runtime.PD](),
				PreferNotRunning[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true, true),
				fakePD("bbb", true, false),
				fakePD("ccc", true, true),
				fakePD("ddd", true, false),
			},
			expected: "bbb",
		},
		{
			desc: "prefer not running",
			ps: []PreferPolicy[*runtime.PD]{
				PreferUnready[*runtime.PD](),
				PreferNotRunning[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true, true),
				fakePD("bbb", true, false),
				fakePD("ccc", false, true),
				fakePD("ddd", true, false),
			},
			expected: "ccc",
		},
		{
			desc: "prefer unready and not running",
			ps: []PreferPolicy[*runtime.PD]{
				PreferUnready[*runtime.PD](),
				PreferNotRunning[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true, true),
				fakePD("bbb", true, false),
				fakePD("ccc", false, true),
				fakePD("ddd", false, false),
			},
			expected: "ddd",
		},
		{
			desc: "prefer priority high - single highest priority",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriorityHigh[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "1"),
				fakePDWithPriority("bbb", true, true, "5"),
				fakePDWithPriority("ccc", true, true, "3"),
				fakePD("ddd", true, true),
			},
			expected: "bbb",
		},
		{
			desc: "prefer priority high - multiple with same highest priority",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriorityHigh[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "5"),
				fakePDWithPriority("bbb", true, true, "5"),
				fakePDWithPriority("ccc", true, true, "3"),
				fakePD("ddd", true, true),
			},
			expected: "aaa",
		},
		{
			desc: "prefer priority high - no priority annotations",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriorityHigh[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true, true),
				fakePD("bbb", true, true),
				fakePD("ccc", true, true),
			},
			expected: "aaa",
		},
		{
			desc: "prefer priority high - invalid priority values ignored",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriorityHigh[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "-1"),
				fakePDWithPriority("bbb", true, true, "invalid"),
				fakePDWithPriority("ccc", true, true, "2"),
				fakePD("ddd", true, true),
			},
			expected: "ccc",
		},
		{
			desc: "prefer priority high - negative priority ignored",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriorityHigh[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "-100"),
				fakePDWithPriority("bbb", true, true, "0"),
				fakePD("ccc", true, true),
			},
			expected: "bbb",
		},
		{
			desc: "prefer priority high combined with prefer unready",
			ps: []PreferPolicy[*runtime.PD]{
				PreferUnready[*runtime.PD](),
				PreferPriorityHigh[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "10"),
				fakePDWithPriority("bbb", true, false, "5"),
				fakePDWithPriority("ccc", true, false, "8"),
				fakePD("ddd", true, false),
			},
			expected: "aaa",
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := NewSelector(c.ps...)
			choosed := s.Choose(c.allowed)
			assert.Equal(tt, c.expected, choosed)
		})
	}
}

func fakePD(name string, running, ready bool) *runtime.PD {
	return runtime.FromPD(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
		obj.Generation = 2
		obj.Labels = map[string]string{
			v1alpha1.LabelKeyInstanceRevisionHash: "test",
		}
		obj.Status.CurrentRevision = "test"
		obj.Status.ObservedGeneration = obj.Generation
		if !running {
			coreutil.SetStatusCondition[scope.PD](obj, *coreutil.NotRunning("", ""))
		}
		if ready {
			coreutil.SetStatusCondition[scope.PD](obj, *coreutil.Ready())
		}
		return obj
	}))
}

func fakePDWithPriority(name string, running, ready bool, priority string) *runtime.PD {
	return runtime.FromPD(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
		obj.Generation = 2
		obj.Labels = map[string]string{
			v1alpha1.LabelKeyInstanceRevisionHash: "test",
		}
		obj.Annotations = map[string]string{
			v1alpha1.AnnoKeyPriority: priority,
		}
		obj.Status.CurrentRevision = "test"
		obj.Status.ObservedGeneration = obj.Generation
		if !running {
			coreutil.SetStatusCondition[scope.PD](obj, *coreutil.NotRunning("", ""))
		}
		if ready {
			coreutil.SetStatusCondition[scope.PD](obj, *coreutil.Ready())
		}
		return obj
	}))
}

func TestPreferPriorityHigh(t *testing.T) {
	cases := []struct {
		desc     string
		input    []*runtime.PD
		expected []string
	}{
		{
			desc: "single instance with highest priority",
			input: []*runtime.PD{
				fakePDWithPriority("pd-0", true, true, "1"),
				fakePDWithPriority("pd-1", true, true, "10"),
				fakePDWithPriority("pd-2", true, true, "5"),
			},
			expected: []string{"pd-1"},
		},
		{
			desc: "multiple instances with same highest priority",
			input: []*runtime.PD{
				fakePDWithPriority("pd-0", true, true, "10"),
				fakePDWithPriority("pd-1", true, true, "10"),
				fakePDWithPriority("pd-2", true, true, "5"),
			},
			expected: []string{"pd-0", "pd-1"},
		},
		{
			desc: "no priority annotations returns all",
			input: []*runtime.PD{
				fakePD("pd-0", true, true),
				fakePD("pd-1", true, true),
				fakePD("pd-2", true, true),
			},
			expected: []string{"pd-0", "pd-1", "pd-2"},
		},
		{
			desc: "invalid priorities are ignored",
			input: []*runtime.PD{
				fakePDWithPriority("pd-0", true, true, "invalid"),
				fakePDWithPriority("pd-1", true, true, "3.14"),
				fakePDWithPriority("pd-2", true, true, "5"),
			},
			expected: []string{"pd-2"},
		},
		{
			desc: "negative priorities are ignored",
			input: []*runtime.PD{
				fakePDWithPriority("pd-0", true, true, "-1"),
				fakePDWithPriority("pd-1", true, true, "-100"),
				fakePDWithPriority("pd-2", true, true, "0"),
			},
			expected: []string{"pd-2"},
		},
		{
			desc: "mix of valid, invalid, and missing priorities",
			input: []*runtime.PD{
				fakePD("pd-0", true, true),
				fakePDWithPriority("pd-1", true, true, "invalid"),
				fakePDWithPriority("pd-2", true, true, "3"),
				fakePDWithPriority("pd-3", true, true, "-5"),
				fakePDWithPriority("pd-4", true, true, "3"),
			},
			expected: []string{"pd-2", "pd-4"},
		},
		{
			desc: "all invalid or missing priorities returns all",
			input: []*runtime.PD{
				fakePD("pd-0", true, true),
				fakePDWithPriority("pd-1", true, true, "invalid"),
				fakePDWithPriority("pd-2", true, true, "-1"),
			},
			expected: []string{"pd-0", "pd-1", "pd-2"},
		},
		{
			desc: "priority zero is valid",
			input: []*runtime.PD{
				fakePDWithPriority("pd-0", true, true, "0"),
				fakePDWithPriority("pd-1", true, true, "0"),
			},
			expected: []string{"pd-0", "pd-1"},
		},
		{
			desc: "large priority values",
			input: []*runtime.PD{
				fakePDWithPriority("pd-0", true, true, "1000000"),
				fakePDWithPriority("pd-1", true, true, "999999"),
				fakePDWithPriority("pd-2", true, true, "1000000"),
			},
			expected: []string{"pd-0", "pd-2"},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			policy := PreferPriorityHigh[*runtime.PD]()
			result := policy.Prefer(c.input)

			// Extract names from result
			var resultNames []string
			for _, pd := range result {
				resultNames = append(resultNames, pd.GetName())
			}

			assert.ElementsMatch(tt, c.expected, resultNames)
		})
	}
}
