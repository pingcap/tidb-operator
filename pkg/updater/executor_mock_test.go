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

// MockActor implements the Actor interface for testing
// It provides configurable behavior without reimplementing business logic
type MockActor struct {
	Actions []action

	scaleOutReturnsError   bool
	updateReturnsError     bool
	scaleInUpdateError     bool
	scaleInOutdatedError   bool
	cleanupReturnsError    bool
	cancelOffliningSuccess bool

	online           int
	beingOffline     int
	offlineCompleted int
}

func (m *MockActor) ScaleOut(_ context.Context) (act action, err error) {
	act = actionNone
	defer func() {
		m.Actions = append(m.Actions, act)
	}()
	if m.scaleOutReturnsError {
		return act, assert.AnError
	}

	if m.beingOffline > 0 && m.cancelOffliningSuccess {
		act = actionCancelOffline
		m.beingOffline--
		return act, nil
	} else {
		act = actionScaleOut
	}

	return act, nil
}

func (m *MockActor) Update(_ context.Context) error {
	if m.updateReturnsError {
		return assert.AnError
	}
	m.Actions = append(m.Actions, actionUpdate)
	return nil
}

func (m *MockActor) ScaleInUpdate(_ context.Context) (act action, unavailable bool, err error) {
	act = actionNone
	defer func() {
		m.Actions = append(m.Actions, act)
	}()
	if m.scaleInUpdateError {
		return act, false, assert.AnError
	}

	if m.offlineCompleted > 0 {
		m.offlineCompleted--
		return actionDeleteOffline, true, nil
	}
	if m.beingOffline > 0 {
		return act, false, nil
	}
	if m.online > 0 {
		act = actionSetOffline
		m.beingOffline++
		m.online--
		return act, false, nil
	}

	return act, false, nil
}

func (m *MockActor) ScaleInOutdated(_ context.Context) (act action, unavailable bool, err error) {
	act = actionNone
	defer func() {
		m.Actions = append(m.Actions, act)
	}()
	if m.scaleInOutdatedError {
		return actionNone, false, assert.AnError
	}

	if m.offlineCompleted > 0 {
		act = actionDeleteOffline
		m.offlineCompleted--
		return act, true, nil
	}
	if m.beingOffline > 0 {
		return act, false, nil
	}
	if m.online > 0 {
		act = actionSetOffline
		m.beingOffline++
		m.online--
		return act, false, nil
	}

	return act, false, nil
}

func (m *MockActor) Cleanup(_ context.Context) error {
	if m.cleanupReturnsError {
		return assert.AnError
	}
	m.Actions = append(m.Actions, actionCleanup)
	return nil
}

func TestExecutorWithMock(t *testing.T) {
	tests := []struct {
		name                string
		update              int
		outdated            int
		online              int
		beingOffline        int
		offlineCompleted    int
		desired             int
		unavailableUpdate   int
		unavailableOutdated int
		maxSurge            int
		maxUnavailable      int

		// Mock configuration
		cancelOffliningSuccess bool

		// Mock error simulation
		scaleOutReturnsError bool
		updateReturnsError   bool
		scaleInUpdateError   bool
		scaleInOutdatedError bool
		cleanupReturnsError  bool

		expectedActions []action
		expectedWait    bool
		expectedError   bool
	}{
		{
			name:    "when scale in, set it offline first",
			update:  3,
			desired: 1,
			online:  3,
			expectedActions: []action{
				actionSetOffline,
				actionNone,
			},
			expectedWait: true,
		},
		{
			name:              "when scale in, wait for offline complete",
			update:            3,
			unavailableUpdate: 1,
			beingOffline:      1,
			desired:           1,
			online:            2,
			expectedActions: []action{
				actionNone,
			},
			expectedWait: true,
		},
		{
			name:              "when offline complete, delete it",
			update:            3,
			unavailableUpdate: 1,
			beingOffline:      0,
			offlineCompleted:  1,
			online:            2,
			desired:           1,
			expectedActions: []action{
				actionDeleteOffline,
				actionSetOffline,
				actionNone,
			},
			expectedWait: true,
		},
		{
			name:              "scale in to 0",
			update:            3,
			unavailableUpdate: 3,
			beingOffline:      3,
			desired:           0,
			expectedActions: []action{
				actionNone,
			},
			expectedWait: true,
		},
		{
			name:         "rolling update with 0 maxSurge and 1 maxUnavailable",
			update:       0,
			outdated:     3,
			beingOffline: 0,
			desired:      3,
			expectedActions: []action{
				actionUpdate,
			},
			expectedWait: true,
		},
		{
			name:         "scale in and rolling update at same time with 0 maxSurge and 1 maxUnavailable(0)",
			update:       0,
			outdated:     5,
			online:       5,
			beingOffline: 0,
			desired:      3,
			expectedActions: []action{
				actionSetOffline,
				actionNone,
			},
			expectedWait: true,
		},
		{
			name:                "scale in and rolling update at same time with 0 maxSurge and 1 maxUnavailable(1)",
			update:              0,
			outdated:            5,
			unavailableOutdated: 1,
			online:              4,
			offlineCompleted:    1,
			desired:             3,
			expectedActions: []action{
				actionDeleteOffline,
				actionSetOffline,
				actionNone,
			},
			expectedWait: true,
		},
		{
			name:           "no operation needed",
			update:         3,
			outdated:       0,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []action{
				actionCleanup,
			},
			expectedWait: false,
		},
		{
			name:              "all instances unavailable",
			update:            3,
			online:            3,
			desired:           2,
			unavailableUpdate: 3,
			maxSurge:          0,
			maxUnavailable:    1,
			expectedActions: []action{
				actionSetOffline,
				actionNone,
			},
			expectedWait: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := &MockActor{
				beingOffline:           tt.beingOffline,
				offlineCompleted:       tt.offlineCompleted,
				online:                 tt.online,
				cancelOffliningSuccess: tt.cancelOffliningSuccess,
				scaleOutReturnsError:   tt.scaleOutReturnsError,
				updateReturnsError:     tt.updateReturnsError,
				scaleInUpdateError:     tt.scaleInUpdateError,
				scaleInOutdatedError:   tt.scaleInOutdatedError,
				cleanupReturnsError:    tt.cleanupReturnsError,
			}

			e := NewExecutor(act, tt.update, tt.outdated, tt.beingOffline, tt.offlineCompleted, tt.desired,
				tt.unavailableUpdate, tt.unavailableOutdated, 0, 1)

			wait, err := e.Do(context.TODO())

			if tt.expectedError {
				require.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error: %v", err)
				assert.Equal(t, tt.expectedWait, wait, "wait result mismatch")
			}

			assert.Equal(t, tt.expectedActions, act.Actions, "actions mismatch")
		})
	}
}
