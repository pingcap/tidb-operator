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
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
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
				fakePD("aaa", true),
				fakePD("bbb", false),
				fakePD("ccc", true),
				fakePD("ddd", false),
			},
			expected: "aaa",
		},
		{
			desc: "prefer unavailable",
			ps: []PreferPolicy[*runtime.PD]{
				PreferUnready[*runtime.PD](),
			},
			allowed: []*runtime.PD{
				fakePD("aaa", true),
				fakePD("bbb", false),
				fakePD("ccc", true),
				fakePD("ddd", false),
			},
			expected: "bbb",
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

func fakePD(name string, ready bool) *runtime.PD {
	return runtime.FromPD(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
		obj.Generation = 2
		obj.Labels = map[string]string{
			v1alpha1.LabelKeyInstanceRevisionHash: "test",
		}
		obj.Status.CurrentRevision = "test"
		coreutil.SetStatusCondition[scope.PD](obj, *coreutil.Ready())
		if ready {
			obj.Status.ObservedGeneration = obj.Generation
		}
		return obj
	}))
}
