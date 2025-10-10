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
)

type FakeActor struct {
	Actions []action

	unavailableUpdate   int
	unavailableOutdated int
	update              int
	outdated            int

	preferAvailable bool
}

func (a *FakeActor) ScaleOut(_ context.Context) error {
	a.Actions = append(a.Actions, actionScaleOut)
	return nil
}

func (a *FakeActor) ScaleInOutdated(_ context.Context) (unavailable bool, err error) {
	a.Actions = append(a.Actions, actionScaleInOutdated)
	a.outdated -= 1
	if a.preferAvailable || a.unavailableOutdated == 0 {
		return false, nil
	}

	a.unavailableOutdated -= 1
	return true, nil
}

func (a *FakeActor) ScaleInUpdate(_ context.Context) (unavailable bool, err error) {
	a.Actions = append(a.Actions, actionScaleInUpdate)
	a.update -= 1
	if a.preferAvailable || a.unavailableUpdate == 0 {
		return false, nil
	}

	a.unavailableUpdate -= 1
	return true, nil
}

func (a *FakeActor) Update(_ context.Context) error {
	a.Actions = append(a.Actions, actionUpdate)
	return nil
}

func (a *FakeActor) Cleanup(_ context.Context) error {
	a.Actions = append(a.Actions, actionCleanup)
	return nil
}

func (a *FakeActor) RecordedActions() []action {
	return a.Actions
}

func TestExecutor(t *testing.T) {
	cases := []struct {
		desc                string
		update              int
		outdated            int
		desired             int
		unavailableUpdate   int
		unavailableOutdated int
		maxSurge            int
		maxUnavailable      int

		preferAvailable bool

		expectedActions []action
		expectedWait    bool
	}{
		{
			desc:           "do nothing",
			update:         3,
			outdated:       0,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionCleanup,
			},
		},
		{
			desc:           "scale out from 0 with 0 maxSurge",
			update:         0,
			outdated:       0,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleOut,
				actionScaleOut,
				actionScaleOut,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out from 0 with 1 maxSurge",
			update:         0,
			outdated:       0,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleOut,
				actionScaleOut,
				actionScaleOut,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in to 0",
			update:         3,
			outdated:       0,
			desired:        0,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleInUpdate,
				actionScaleInUpdate,
				actionScaleInUpdate,
				actionCleanup,
			},
		},
		{
			desc:           "rolling update with 0 maxSurge and 1 maxUnavailable(0)",
			update:         0,
			outdated:       3,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 0 maxSurge and 1 maxUnavailable(1)",
			update:         1,
			outdated:       2,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 0 maxSurge and 1 maxUnavailable(2)",
			update:         2,
			outdated:       1,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 1 maxSurge and 0 maxUnavailable(0)",
			update:         0,
			outdated:       3,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleOut,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 1 maxSurge and 0 maxUnavailable(1)",
			update:         1,
			outdated:       3,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 1 maxSurge and 0 maxUnavailable(2)",
			update:         2,
			outdated:       2,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 1 maxSurge and 0 maxUnavailable(3)",
			update:         3,
			outdated:       1,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleInOutdated,
				actionCleanup,
			},
		},
		{
			desc:           "rolling update with 1 maxSurge and 1 maxUnavailable(0)",
			update:         0,
			outdated:       3,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleOut,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			// when 1 update is ready but 1 is not
			desc:              "rolling update with 1 maxSurge and 1 maxUnavailable and 1 unavailableUpdate(0-1)",
			update:            2,
			outdated:          2,
			desired:           3,
			unavailableUpdate: 1,
			maxSurge:          1,
			maxUnavailable:    1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			// there is still 1 update not ready
			desc:              "rolling update with 1 maxSurge and 1 maxUnavailable and 1 unavailableUpdate(0-2)",
			update:            3,
			outdated:          1,
			desired:           3,
			unavailableUpdate: 1,
			maxSurge:          1,
			maxUnavailable:    1,
			expectedActions: []action{
				actionScaleInOutdated,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 1 maxSurge and 1 maxUnavailable(1)",
			update:         2,
			outdated:       2,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
				actionScaleInOutdated,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 2 maxSurge and 0 maxUnavailable(0)",
			update:         0,
			outdated:       3,
			desired:        3,
			maxSurge:       2,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleOut,
				actionScaleOut,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 2 maxSurge and 0 maxUnavailable(1)",
			update:         2,
			outdated:       3,
			desired:        3,
			maxSurge:       2,
			maxUnavailable: 0,
			expectedActions: []action{
				actionUpdate,
				actionScaleInOutdated,
			},
			expectedWait: true,
		},
		{
			desc:           "rolling update with 2 maxSurge and 0 maxUnavailable(2)",
			update:         3,
			outdated:       1,
			desired:        3,
			maxSurge:       2,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleInOutdated,
				actionCleanup,
			},
		},
		{
			desc:           "scale out and rolling update at same time with 0 maxSurge and 1 maxUnavailable(0)",
			update:         0,
			outdated:       3,
			desired:        5,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleOut,
				actionScaleOut,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 0 maxSurge and 1 maxUnavailable(1)",
			update:         2,
			outdated:       3,
			desired:        5,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 0 maxSurge and 1 maxUnavailable(2)",
			update:         3,
			outdated:       2,
			desired:        5,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 0 maxSurge and 1 maxUnavailable(3)",
			update:         4,
			outdated:       1,
			desired:        5,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 1 maxSurge and 0 maxUnavailable(0)",
			update:         0,
			outdated:       3,
			desired:        5,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleOut,
				actionScaleOut,
				actionScaleOut,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 1 maxSurge and 0 maxUnavailable(1)",
			update:         3,
			outdated:       3,
			desired:        5,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 1 maxSurge and 0 maxUnavailable(2)",
			update:         4,
			outdated:       2,
			desired:        5,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 1 maxSurge and 0 maxUnavailable(3)",
			update:         5,
			outdated:       1,
			desired:        5,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleInOutdated,
				actionCleanup,
			},
		},
		{
			desc:           "scale out and rolling update at same time with 1 maxSurge and 1 maxUnavailable(0)",
			update:         0,
			outdated:       3,
			desired:        5,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleOut,
				actionScaleOut,
				actionScaleOut,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 1 maxSurge and 1 maxUnavailable(1)",
			update:         3,
			outdated:       3,
			desired:        5,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale out and rolling update at same time with 1 maxSurge and 1 maxUnavailable(2)",
			update:         5,
			outdated:       1,
			desired:        5,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleInOutdated,
				actionCleanup,
			},
		},
		{
			desc:           "scale in and rolling update at same time with 0 maxSurge and 1 maxUnavailable(0)",
			update:         0,
			outdated:       5,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleInOutdated,
				actionScaleInOutdated,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in and rolling update at same time with 0 maxSurge and 1 maxUnavailable(1)",
			update:         1,
			outdated:       2,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in and rolling update at same time with 0 maxSurge and 1 maxUnavailable(2)",
			update:         2,
			outdated:       1,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in and rolling update at same time with 1 maxSurge and 0 maxUnavailable(0)",
			update:         0,
			outdated:       5,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleInOutdated,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in and rolling update at same time with 1 maxSurge and 0 maxUnavailable(1)",
			update:         1,
			outdated:       3,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in and rolling update at same time with 1 maxSurge and 0 maxUnavailable(2)",
			update:         2,
			outdated:       2,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in and rolling update at same time with 1 maxSurge and 0 maxUnavailable(3)",
			update:         3,
			outdated:       1,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 0,
			expectedActions: []action{
				actionScaleInOutdated,
				actionCleanup,
			},
		},
		{
			desc:           "scale in and rolling update at same time with 1 maxSurge and 1 maxUnavailable(0)",
			update:         0,
			outdated:       5,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionScaleInOutdated,
				actionUpdate,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "scale in and rolling update at same time with 1 maxSurge and 1 maxUnavailable(1)",
			update:         2,
			outdated:       2,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionUpdate,
				actionScaleInOutdated,
			},
			expectedWait: true,
		},
		{
			desc:                "rolling update with all are unavailable(0)",
			update:              0,
			outdated:            3,
			desired:             3,
			unavailableOutdated: 3,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
				actionUpdate,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:              "scale in when all are unavailable(0)",
			update:            5,
			outdated:          0,
			desired:           3,
			unavailableUpdate: 5,
			maxSurge:          0,
			maxUnavailable:    1,
			expectedActions: []action{
				actionScaleInUpdate,
				actionScaleInUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(0)",
			update:              1,
			outdated:            4,
			desired:             3,
			unavailableOutdated: 2,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionScaleInOutdated,
				actionScaleInOutdated,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(1-0)",
			update:              1,
			outdated:            3,
			desired:             3,
			unavailableOutdated: 2,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionScaleInOutdated,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(1-1)",
			update:              1,
			outdated:            3,
			desired:             3,
			unavailableOutdated: 1,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionScaleInOutdated,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(2-0)",
			update:              1,
			outdated:            2,
			desired:             3,
			unavailableOutdated: 2,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(2-1)",
			update:              1,
			outdated:            2,
			desired:             3,
			unavailableOutdated: 1,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(2-2)",
			update:              1,
			outdated:            2,
			desired:             3,
			unavailableOutdated: 0,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions:     nil,
			expectedWait:        true,
		},
		{
			desc:                "complex case(3-0)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableOutdated: 1,
			unavailableUpdate:   2,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(3-1)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableOutdated: 0,
			unavailableUpdate:   2,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions:     nil,
			expectedWait:        true,
		},
		{
			desc:                "complex case(3-2)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableOutdated: 1,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(3-3)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableOutdated: 0,
			unavailableUpdate:   1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions:     nil,
			expectedWait:        true,
		},
		{
			desc:                "complex case(3-4)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableOutdated: 1,
			unavailableUpdate:   0,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "complex case(3-5)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableOutdated: 0,
			unavailableUpdate:   0,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "rolling update with one is unavailable(0)",
			update:              0,
			outdated:            3,
			desired:             3,
			unavailableOutdated: 1,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "rolling update with one is unavailable(1)",
			update:              1,
			outdated:            2,
			desired:             3,
			unavailableUpdate:   1,
			unavailableOutdated: 0,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedWait:        true,
		},
		{
			desc:                "rolling update with one is unavailable(2)",
			update:              1,
			outdated:            2,
			desired:             3,
			unavailableUpdate:   0,
			unavailableOutdated: 0,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "rolling update with two are unavailable(0)",
			update:              0,
			outdated:            3,
			desired:             3,
			unavailableUpdate:   0,
			unavailableOutdated: 2,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "rolling update with two are unavailable(1)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableUpdate:   2,
			unavailableOutdated: 0,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedWait:        true,
		},
		{
			desc:                "rolling update with two are unavailable(2)",
			update:              2,
			outdated:            1,
			desired:             3,
			unavailableUpdate:   0,
			unavailableOutdated: 0,
			maxSurge:            0,
			maxUnavailable:      1,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:                "outdated is unavailable",
			update:              1,
			outdated:            1,
			desired:             1,
			unavailableUpdate:   0,
			unavailableOutdated: 1,
			maxSurge:            1,
			maxUnavailable:      0,
			expectedActions: []action{
				actionScaleInOutdated,
				actionCleanup,
			},
			expectedWait: false,
		},
		{
			desc:                "2 outdated are both unavailable",
			update:              2,
			outdated:            2,
			desired:             2,
			unavailableUpdate:   0,
			unavailableOutdated: 2,
			maxSurge:            1,
			maxUnavailable:      0,
			expectedActions: []action{
				actionScaleInOutdated,
				actionScaleInOutdated,
				actionCleanup,
			},
			expectedWait: false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			act := &FakeActor{
				update:              c.update,
				outdated:            c.outdated,
				unavailableUpdate:   c.unavailableUpdate,
				unavailableOutdated: c.unavailableOutdated,
			}
			e := NewExecutor(act, c.update, c.outdated, c.desired, c.unavailableUpdate, c.unavailableOutdated, c.maxSurge, c.maxUnavailable)
			wait, err := e.Do(context.TODO())
			require.NoError(tt, err)
			assert.Equal(tt, c.expectedWait, wait, c.desc)
			assert.Equal(tt, c.expectedActions, act.Actions, c.desc)
		})
	}
}
