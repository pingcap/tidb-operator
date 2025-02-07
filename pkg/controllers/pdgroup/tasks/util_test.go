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

package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestHeadlessServiceName(t *testing.T) {
	cases := []struct {
		desc         string
		groupName    string
		expectedName string
	}{
		{
			desc:         "normal",
			groupName:    "xxx",
			expectedName: "xxx-pd-peer",
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			assert.Equal(tt, c.expectedName, HeadlessServiceName(c.groupName), c.desc)
		})
	}
}

func TestNotLeaderPolicy(t *testing.T) {
	cases := []struct {
		desc     string
		pds      []*v1alpha1.PD
		expected []*v1alpha1.PD
	}{
		{
			desc: "none",
		},
		{
			desc: "no leader",
			pds: []*v1alpha1.PD{
				fake.FakeObj[v1alpha1.PD]("aaa"),
				fake.FakeObj[v1alpha1.PD]("bbb"),
				fake.FakeObj[v1alpha1.PD]("ccc"),
			},
			expected: []*v1alpha1.PD{
				fake.FakeObj[v1alpha1.PD]("aaa"),
				fake.FakeObj[v1alpha1.PD]("bbb"),
				fake.FakeObj[v1alpha1.PD]("ccc"),
			},
		},
		{
			desc: "filter leader",
			pds: []*v1alpha1.PD{
				fake.FakeObj[v1alpha1.PD]("aaa"),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.IsLeader = true
					return obj
				}),
				fake.FakeObj[v1alpha1.PD]("ccc"),
			},
			expected: []*v1alpha1.PD{
				fake.FakeObj[v1alpha1.PD]("aaa"),
				fake.FakeObj[v1alpha1.PD]("ccc"),
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			policy := NotLeaderPolicy()
			prefer := policy.Prefer(runtime.FromPDSlice(c.pds))
			assert.Equal(tt, c.expected, runtime.ToPDSlice(prefer), c.desc)
		})
	}
}
