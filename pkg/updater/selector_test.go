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
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestPreferNewer(t *testing.T) {
	older := fakePD("older", true, true)
	older.SetCreationTimestamp(metav1.NewTime(time.Unix(100, 0)))
	newer := fakePD("newer", true, true)
	newer.SetCreationTimestamp(metav1.NewTime(time.Unix(200, 0)))
	tied := fakePD("tied", true, true)
	tied.SetCreationTimestamp(newer.GetCreationTimestamp())
	unknown := fakePD("unknown", true, true)
	unknown2 := fakePD("unknown2", true, true)

	cases := []struct {
		name            string
		input, expected []*runtime.PD
	}{
		{name: "empty"},
		{name: "single", input: []*runtime.PD{older}, expected: []*runtime.PD{older}},
		{name: "newest last", input: []*runtime.PD{older, newer}, expected: []*runtime.PD{newer}},
		{name: "newest first", input: []*runtime.PD{newer, older}, expected: []*runtime.PD{newer}},
		{name: "equal timestamps", input: []*runtime.PD{newer, older, tied}, expected: []*runtime.PD{newer, tied}},
		{name: "missing timestamp", input: []*runtime.PD{unknown, newer}, expected: []*runtime.PD{newer}},
		{name: "all timestamps missing", input: []*runtime.PD{unknown, unknown2}, expected: []*runtime.PD{unknown, unknown2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := append([]*runtime.PD(nil), tc.input...)
			assert.Equal(t, tc.expected, PreferNewer[*runtime.PD]().Prefer(tc.input))
			assert.Equal(t, original, tc.input)
		})
	}
}

func TestSelector(t *testing.T) {
	cases := []struct {
		desc     string
		filters  []FilterPolicy[*runtime.PD]
		ps       []PreferPolicy[*runtime.PD]
		allowed  []*runtime.PD
		expected string
	}{
		{
			desc: "filter rejects when no eligible instance",
			filters: []FilterPolicy[*runtime.PD]{
				FilterPolicyFunc[*runtime.PD](func([]*runtime.PD) []*runtime.PD {
					return []*runtime.PD{}
				}),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true, true),
			},
			expected: "",
		},
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
			desc: "prefer priority - single priority",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriority[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "1"),
				fakePDWithPriority("bbb", true, true, "5"),
				fakePDWithPriority("ccc", true, true, "3"),
				fakePD("ddd", true, true),
			},
			expected: "aaa",
		},
		{
			desc: "prefer priority - multiple with same priority",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriority[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "0"),
				fakePDWithPriority("bbb", true, true, "0"),
				fakePDWithPriority("ccc", true, true, "3"),
				fakePD("ddd", true, true),
			},
			expected: "aaa",
		},
		{
			desc: "prefer priority - no priority annotations",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriority[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true, true),
				fakePD("bbb", true, true),
				fakePD("ccc", true, true),
			},
			expected: "aaa",
		},
		{
			desc: "prefer priority - invalid priority values ignored",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriority[*runtime.PD](),
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
			desc: "prefer priority - negative priority ignored",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriority[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "-100"),
				fakePDWithPriority("bbb", true, true, "0"),
				fakePD("ccc", true, true),
			},
			expected: "bbb",
		},
		{
			desc: "prefer priority combined with prefer unready",
			ps: []PreferPolicy[*runtime.PD]{
				PreferUnready[*runtime.PD](),
				PreferPriority[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "0"),
				fakePDWithPriority("bbb", true, false, "5"),
				fakePDWithPriority("ccc", true, false, "8"),
				fakePD("ddd", true, false),
			},
			expected: "aaa",
		},
		{
			desc: "priority, unready, not running",
			ps: []PreferPolicy[*runtime.PD]{
				PreferPriority[*runtime.PD](),
				PreferUnready[*runtime.PD](),
				PreferNotRunning[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePDWithPriority("aaa", true, true, "0"),
				fakePDWithPriority("bbb", true, false, "10"),
				fakePDWithPriority("ccc", false, true, "10"),
				fakePDWithPriority("ddd", false, false, "10"),
				fakePDWithPriority("eee", false, false, "5"),
			},
			expected: "eee",
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := NewSelectorWithFilter(c.filters, c.ps...)
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

func TestPreferPriority(t *testing.T) {
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
			expected: []string{"pd-0"},
		},
		{
			desc: "multiple instances with same highest priority",
			input: []*runtime.PD{
				fakePDWithPriority("pd-0", true, true, "10"),
				fakePDWithPriority("pd-1", true, true, "10"),
				fakePDWithPriority("pd-2", true, true, "5"),
			},
			expected: []string{"pd-2"},
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
				fakePDWithPriority("pd-2", true, true, "999999"),
			},
			expected: []string{"pd-1", "pd-2"},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			policy := PreferPriority[*runtime.PD]()
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

func TestFilterOutdated(t *testing.T) {
	t.Parallel()

	const rev = "rev-current"
	filter := FilterOutdated[*runtime.PD](rev)

	current := fakePD("pd-a", true, true)
	current.Labels = map[string]string{
		v1alpha1.LabelKeyInstanceRevisionHash: rev,
	}
	outdated := fakePD("pd-old", true, true)
	outdated.Labels = map[string]string{
		v1alpha1.LabelKeyInstanceRevisionHash: "rev-old",
	}

	assert.Equal(t, []*runtime.PD{current}, filter.Filter([]*runtime.PD{current, outdated}))
	assert.Empty(t, filter.Filter([]*runtime.PD{outdated}))
}

func TestFilterOutdatedCancelOfflineSelector(t *testing.T) {
	t.Parallel()

	const rev = "rev-current"
	selector := NewSelectorWithFilter(
		[]FilterPolicy[*runtime.PD]{
			FilterOutdated[*runtime.PD](rev),
		},
	)

	current := fakePD("pd-a", true, true)
	current.Labels = map[string]string{
		v1alpha1.LabelKeyInstanceRevisionHash: rev,
	}
	outdated := fakePD("pd-old", true, true)
	outdated.Labels = map[string]string{
		v1alpha1.LabelKeyInstanceRevisionHash: "rev-old",
	}

	assert.Equal(t, "pd-a", selector.Choose([]*runtime.PD{current, outdated}))
	assert.Empty(t, selector.Choose([]*runtime.PD{outdated}))
}

func TestFilterAnnotationAbsent(t *testing.T) {
	t.Parallel()

	filter := FilterAnnotationAbsent[*runtime.TiProxy](v1alpha1.AnnoKeyTiProxyReviveAbandoned)

	revivable := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tiproxy-a",
		},
	}
	abandoned := &runtime.TiProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tiproxy-b",
			Annotations: map[string]string{
				v1alpha1.AnnoKeyTiProxyReviveAbandoned: v1alpha1.AnnoValTrue,
			},
		},
	}

	assert.Equal(t, []*runtime.TiProxy{revivable}, filter.Filter([]*runtime.TiProxy{revivable, abandoned}))
	assert.Empty(t, filter.Filter([]*runtime.TiProxy{abandoned}))
}

func TestPreferVolumeCapacityExceedsRequest(t *testing.T) {
	makeInstance := func(name string, status metav1.ConditionStatus, generation int64) *runtime.PD {
		in := fakePD(name, true, true)
		in.Generation = 7
		if status != "" {
			coreutil.SetStatusCondition[scope.PD](runtime.ToPD(in), metav1.Condition{
				Type: v1alpha1.CondVolumeCapacityExceedsRequest, Status: status, Reason: "CapacityObserved",
			})
			for i := range in.Status.Conditions {
				if in.Status.Conditions[i].Type == v1alpha1.CondVolumeCapacityExceedsRequest {
					in.Status.Conditions[i].ObservedGeneration = generation
				}
			}
		}
		return in
	}
	absent := makeInstance("absent", "", 7)
	current := makeInstance("current", metav1.ConditionTrue, 7)
	other := makeInstance("other", metav1.ConditionTrue, 7)
	stale := makeInstance("stale", metav1.ConditionTrue, 6)
	matched := makeInstance("matched", metav1.ConditionFalse, 7)
	unknown := makeInstance("unknown", metav1.ConditionUnknown, 7)
	policy := PreferVolumeCapacityExceedsRequest[*runtime.PD]()
	assert.Equal(t, []*runtime.PD{current, other}, policy.Prefer([]*runtime.PD{absent, stale, matched, current, unknown, other}))
	assert.Empty(t, policy.Prefer([]*runtime.PD{absent, stale, matched, unknown}))
	selector := NewSelector(PreferPriority[*runtime.PD](), PreferUnready[*runtime.PD](), PreferNotRunning[*runtime.PD](), policy)
	absent.Annotations = map[string]string{v1alpha1.AnnoKeyPriority: "0"}
	assert.Equal(t, "current", selector.Choose([]*runtime.PD{absent, current}))
	assert.Equal(t, "absent", selector.Choose([]*runtime.PD{absent, stale}))
	other.Annotations = map[string]string{v1alpha1.AnnoKeyPriority: "0"}
	assert.Equal(t, "other", selector.Choose([]*runtime.PD{current, other}))
	// A group-specific policy appended by the builder retains higher priority.
	topology := PreferPolicyFunc[*runtime.PD](func(in []*runtime.PD) []*runtime.PD { return in[:1] })
	assert.Equal(t, "absent", NewSelector(policy, topology).Choose([]*runtime.PD{absent, current}))
}
