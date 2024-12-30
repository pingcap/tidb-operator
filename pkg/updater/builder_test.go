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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestBuilder(t *testing.T) {
	addHook := AddHookFunc[*runtime.PD](func(pd *runtime.PD) *runtime.PD { return pd })
	updateHook := UpdateHookFunc[*runtime.PD](func(update, _ *runtime.PD) *runtime.PD { return update })
	delHook := DelHookFunc[*runtime.PD](func(_ string) {})
	pd0 := fake.FakeObj[v1alpha1.PD]("pd-0")
	pd1 := fake.FakeObj[v1alpha1.PD]("pd-1")
	cli := client.NewFakeClient(pd0, pd1)

	cases := []struct {
		desc         string
		desired      int
		revision     string
		expectedWait bool
	}{
		{
			desc:         "update",
			desired:      2,
			revision:     "v1",
			expectedWait: true,
		},
		{
			desc:         "scale out",
			desired:      3,
			expectedWait: false,
		},
		{
			desc:         "scale in",
			desired:      1,
			expectedWait: false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			bld := New[*runtime.PD]().
				WithInstances(fake.FakeObj[runtime.PD]("pd-0"), fake.FakeObj[runtime.PD]("pd-1")).WithDesired(c.desired).
				WithMaxSurge(1).
				WithMaxUnavailable(1).
				WithRevision(c.revision).
				WithClient(cli).
				WithNewFactory(NewFunc[*runtime.PD](func() *runtime.PD { return &runtime.PD{} })).
				WithAddHooks(addHook).
				WithUpdateHooks(updateHook).
				WithDelHooks(delHook).Build()
			wait, err := bld.Do(context.Background())
			require.NoError(t, err)
			assert.Equal(t, c.expectedWait, wait)
		})
	}
}
