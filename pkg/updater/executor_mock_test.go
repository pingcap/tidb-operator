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

// MockAction represents actions performed by MockActor
type MockAction int

const (
	MockActionScaleOut MockAction = iota
	MockActionUpdate
	MockActionScaleInUpdate
	MockActionScaleInOutdated
	MockActionCleanup
	MockActionCancelOfflining
	MockActionSetOffline
	MockActionDeleteOfflined
)

// MockActor implements the Actor interface for testing
// It provides configurable behavior without reimplementing business logic
type MockActor struct {
	Actions []MockAction

	// Configuration
	enableTwoStep bool

	// Behavior flags
	scaleOutReturnsCreated  bool
	scaleOutReturnsError    bool
	updateReturnsError      bool
	scaleInUpdateOperated   bool
	scaleInUpdateError      bool
	scaleInOutdatedOperated bool
	scaleInOutdatedError    bool
	cleanupReturnsError     bool

	// Two-step behavior simulation
	hasOffliningInstance   bool
	hasOfflinedInstance    bool
	scaleInShouldWait      bool
	cancelOffliningSuccess bool

	// Call counts
	scaleOutCount        int
	updateCount          int
	scaleInUpdateCount   int
	scaleInOutdatedCount int
	cleanupCount         int
}

func (m *MockActor) ScaleOut(_ context.Context) (bool, error) {
	if m.scaleOutReturnsError {
		return false, assert.AnError
	}

	m.Actions = append(m.Actions, MockActionScaleOut)
	m.scaleOutCount++

	// Simulate two-step behavior: cancel offlining instead of creating new instance
	if m.enableTwoStep && m.hasOffliningInstance && m.cancelOffliningSuccess {
		// Replace the last action with cancel offlining
		m.Actions[len(m.Actions)-1] = MockActionCancelOfflining
		m.hasOffliningInstance = false
		return false, nil // false indicates no new instance was created
	}

	return m.scaleOutReturnsCreated, nil
}

func (m *MockActor) Update(_ context.Context) error {
	if m.updateReturnsError {
		return assert.AnError
	}

	m.Actions = append(m.Actions, MockActionUpdate)
	m.updateCount++
	return nil
}

func (m *MockActor) ScaleInUpdate(_ context.Context) (operated, unavailable bool, err error) {
	if m.scaleInUpdateError {
		return false, false, assert.AnError
	}

	m.Actions = append(m.Actions, MockActionScaleInUpdate)
	m.scaleInUpdateCount++

	if m.enableTwoStep {
		// Simulate two-step deletion behavior
		if m.hasOfflinedInstance {
			m.Actions[len(m.Actions)-1] = MockActionDeleteOfflined
			m.hasOfflinedInstance = false
			return true, true, nil // Delete offlined instance
		}
		if m.hasOffliningInstance {
			// Wait for offlining to complete
			return false, false, nil
		}
		if m.scaleInShouldWait {
			// Start offline process
			m.Actions[len(m.Actions)-1] = MockActionSetOffline
			m.hasOffliningInstance = true
			return true, true, nil // Was unavailable
		}
	}

	return m.scaleInUpdateOperated, true, nil
}

func (m *MockActor) ScaleInOutdated(_ context.Context) (operated, unavailable bool, err error) {
	if m.scaleInOutdatedError {
		return false, false, assert.AnError
	}

	m.Actions = append(m.Actions, MockActionScaleInOutdated)
	m.scaleInOutdatedCount++

	if m.enableTwoStep {
		// Simulate two-step deletion behavior
		if m.hasOfflinedInstance {
			m.Actions[len(m.Actions)-1] = MockActionDeleteOfflined
			m.hasOfflinedInstance = false
			return true, true, nil // Delete offlined instance
		}
		if m.hasOffliningInstance {
			// Wait for offlining to complete
			return false, false, nil
		}
		if m.scaleInShouldWait {
			// Start offline process
			m.Actions[len(m.Actions)-1] = MockActionSetOffline
			m.hasOffliningInstance = true
			return true, true, nil // Was unavailable
		}
	}

	return m.scaleInOutdatedOperated, true, nil
}

func (m *MockActor) Cleanup(_ context.Context) error {
	if m.cleanupReturnsError {
		return assert.AnError
	}

	m.Actions = append(m.Actions, MockActionCleanup)
	m.cleanupCount++
	return nil
}

func TestExecutorWithMock(t *testing.T) {
	cases := []struct {
		desc                string
		update              int
		outdated            int
		offlining           int
		offlined            int
		desired             int
		unavailableUpdate   int
		unavailableOutdated int
		maxSurge            int
		maxUnavailable      int

		// Mock configuration
		enableTwoStep          bool
		hasOffliningInstance   bool
		hasOfflinedInstance    bool
		scaleInShouldWait      bool
		cancelOffliningSuccess bool

		// Mock error simulation
		scaleOutReturnsError bool
		updateReturnsError   bool
		scaleInUpdateError   bool
		scaleInOutdatedError bool
		cleanupReturnsError  bool

		expectedActions []MockAction
		expectedWait    bool
		expectedError   bool
	}{
		// Basic two-step behavior
		{
			desc:              "two-step enabled uses set offline instead of direct scale in",
			enableTwoStep:     true,
			update:            4,
			outdated:          0,
			desired:           3,
			maxSurge:          0,
			maxUnavailable:    1,
			scaleInShouldWait: true,
			expectedActions: []MockAction{
				MockActionSetOffline, // Start offline process
				MockActionCleanup,    // Cleanup after operation
			},
			expectedWait: false,
		},
		{
			desc:                   "two-step scale out cancels offlining instance",
			enableTwoStep:          true,
			update:                 2,
			outdated:               0,
			offlining:              1,
			desired:                3,
			maxSurge:               1,
			maxUnavailable:         1,
			hasOffliningInstance:   true,
			cancelOffliningSuccess: true,
			expectedActions: []MockAction{
				MockActionCancelOfflining, // Cancel offlining instead of creating new instance
				MockActionScaleOut,        // Still need to scale out to reach desired
			},
			expectedWait: true,
		},
		{
			desc:                "two-step deletes offlined instance",
			enableTwoStep:       true,
			update:              4,
			outdated:            0,
			offlined:            1,
			desired:             3,
			maxSurge:            0,
			maxUnavailable:      1,
			hasOfflinedInstance: true,
			expectedActions: []MockAction{
				MockActionDeleteOfflined, // Delete already offlined instance
				MockActionCleanup,        // Complete the operation
			},
			expectedWait: false,
		},
		{
			desc:                 "two-step waits for offlining to complete",
			enableTwoStep:        true,
			update:               4,
			outdated:             0,
			offlining:            1,
			desired:              3,
			maxSurge:             0,
			maxUnavailable:       1,
			hasOffliningInstance: true,
			expectedActions: []MockAction{
				MockActionScaleInUpdate, // Try to scale in but wait
			},
			expectedWait: true,
		},
		// Rolling update scenarios
		{
			desc:           "rolling update with two-step enabled",
			enableTwoStep:  true,
			update:         1,
			outdated:       2,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []MockAction{
				MockActionScaleOut,
				MockActionUpdate,
			},
			expectedWait: true,
		},
		{
			desc:           "regular mode without two-step",
			enableTwoStep:  false,
			update:         4,
			outdated:       0,
			desired:        3,
			maxSurge:       0,
			maxUnavailable: 1,
			expectedActions: []MockAction{
				MockActionScaleInUpdate,
				MockActionCleanup,
			},
			expectedWait: false,
		},
		// Error scenarios
		{
			desc:                 "two-step scale out fails",
			enableTwoStep:        true,
			update:               1,
			outdated:             2,
			desired:              3,
			maxSurge:             1,
			maxUnavailable:       1,
			scaleOutReturnsError: true,
			expectedActions:      nil,
			expectedError:        true,
		},
		{
			desc:               "two-step scale in fails",
			enableTwoStep:      true,
			update:             4,
			outdated:           0,
			desired:            3,
			maxSurge:           0,
			maxUnavailable:     1,
			scaleInUpdateError: true,
			expectedActions:    nil,
			expectedError:      true,
		},
		// Edge cases
		{
			desc:           "no operation needed",
			enableTwoStep:  true,
			update:         3,
			outdated:       0,
			desired:        3,
			maxSurge:       1,
			maxUnavailable: 1,
			expectedActions: []MockAction{
				MockActionCleanup,
			},
			expectedWait: false,
		},
		{
			desc:              "all instances unavailable",
			enableTwoStep:     true,
			update:            3,
			outdated:          0,
			desired:           2,
			unavailableUpdate: 3,
			maxSurge:          0,
			maxUnavailable:    1,
			scaleInShouldWait: true,
			expectedActions:   []MockAction{MockActionSetOffline},
			expectedWait:      true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			act := &MockActor{
				enableTwoStep:           c.enableTwoStep,
				hasOffliningInstance:    c.hasOffliningInstance,
				hasOfflinedInstance:     c.hasOfflinedInstance,
				scaleInShouldWait:       c.scaleInShouldWait,
				cancelOffliningSuccess:  c.cancelOffliningSuccess,
				scaleOutReturnsCreated:  true, // Default behavior
				scaleInUpdateOperated:   true,
				scaleInOutdatedOperated: true,
				scaleOutReturnsError:    c.scaleOutReturnsError,
				updateReturnsError:      c.updateReturnsError,
				scaleInUpdateError:      c.scaleInUpdateError,
				scaleInOutdatedError:    c.scaleInOutdatedError,
				cleanupReturnsError:     c.cleanupReturnsError,
			}

			e := NewExecutor(act, c.update, c.outdated, c.offlining, c.offlined, c.desired,
				c.unavailableUpdate, c.unavailableOutdated, c.maxSurge, c.maxUnavailable)

			wait, err := e.Do(context.TODO())

			if c.expectedError {
				require.Error(tt, err, "expected error but got none")
			} else {
				require.NoError(tt, err, "unexpected error: %v", err)
				assert.Equal(tt, c.expectedWait, wait, "wait result mismatch")
			}

			assert.Equal(tt, c.expectedActions, act.Actions, "actions mismatch")
		})
	}
}
