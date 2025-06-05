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

package features

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestFeatureGates(t *testing.T) {
	cases := []struct {
		desc string

		obj *v1alpha1.PD

		feat    meta.Feature
		enabled bool
	}{
		{
			desc: "aaa is enabled",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Features = []meta.Feature{"aaa"}
				return obj
			}),
			feat:    meta.Feature("aaa"),
			enabled: true,
		},
		{
			desc: "bbb is not enabled",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Features = []meta.Feature{"aaa"}
				return obj
			}),
			feat:    meta.Feature("bbb"),
			enabled: false,
		},
		{
			desc: "no feature",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			feat:    meta.Feature("aaa"),
			enabled: false,
		},
	}

	for i := range cases {
		c := &cases[i]

		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			fg := New[scope.PD](c.obj)
			assert.Equal(tt, c.enabled, fg.Enabled(c.feat), c.desc)
		})
	}
}
