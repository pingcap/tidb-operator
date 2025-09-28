// Copyright 2025 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package backup

import (
	"encoding/binary"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestQueryAllLogBackupKeys(t *testing.T) {
	bt := &backupTracker{}

	t.Run("all keys exist", func(t *testing.T) {
		// Create mock etcd client
		mockClient := &mockPDEtcdClient{
			getResponse: map[string][]*pdapi.KeyValue{
				"/tidb/br-stream/checkpoint/test-backup": {
					{Value: encodeUint64(123456789)},
				},
				"/tidb/br-stream/info/test-backup": {
					{Value: []byte("info")},
				},
				"/tidb/br-stream/pause/test-backup": {
					{Value: []byte("")}, // V1 pause format (manual)
				},
				"/tidb/br-stream/last-error/test-backup": {},
			},
		}

		// Test batch query
		state, err := bt.queryAllLogBackupKeys(mockClient, "test-backup")
		if err != nil {
			t.Fatalf("queryAllLogBackupKeys failed: %v", err)
		}

		// Verify results
		if state.CheckpointTS != 123456789 {
			t.Errorf("expected CheckpointTS 123456789, got %d", state.CheckpointTS)
		}

		if !state.InfoExists {
			t.Error("expected InfoExists to be true")
		}

		if !state.IsPaused {
			t.Error("expected IsPaused to be true")
		}

		if state.PauseReason != "manual" {
			t.Errorf("expected PauseReason 'manual', got %s", state.PauseReason)
		}

		if state.KernelState != LogBackupKernelPaused {
			t.Errorf("expected KernelState paused, got %s", state.KernelState)
		}
	})

	t.Run("no keys exist", func(t *testing.T) {
		// All keys return empty
		mockClient := &mockPDEtcdClient{
			getResponse: map[string][]*pdapi.KeyValue{
				"/tidb/br-stream/checkpoint/test-backup": {},
				"/tidb/br-stream/info/test-backup":       {},
				"/tidb/br-stream/pause/test-backup":      {},
				"/tidb/br-stream/last-error/test-backup": {},
			},
		}

		state, err := bt.queryAllLogBackupKeys(mockClient, "test-backup")
		if err != nil {
			t.Fatalf("queryAllLogBackupKeys failed: %v", err)
		}

		// Verify defaults
		if state.CheckpointTS != 0 {
			t.Errorf("expected CheckpointTS 0, got %d", state.CheckpointTS)
		}

		if state.InfoExists {
			t.Error("expected InfoExists to be false")
		}

		if state.IsPaused {
			t.Error("expected IsPaused to be false")
		}

		if state.PauseReason != "" {
			t.Errorf("expected empty PauseReason, got %s", state.PauseReason)
		}

		if state.KernelState != LogBackupKernelRunning {
			t.Errorf("expected KernelState running, got %s", state.KernelState)
		}
	})

	t.Run("partial keys missing", func(t *testing.T) {
		// Only checkpoint and info exist
		mockClient := &mockPDEtcdClient{
			getResponse: map[string][]*pdapi.KeyValue{
				"/tidb/br-stream/checkpoint/test-backup": {
					{Value: encodeUint64(999999)},
				},
				"/tidb/br-stream/info/test-backup": {
					{Value: []byte("info")},
				},
				"/tidb/br-stream/pause/test-backup":      {},
				"/tidb/br-stream/last-error/test-backup": {},
			},
		}

		state, err := bt.queryAllLogBackupKeys(mockClient, "test-backup")
		if err != nil {
			t.Fatalf("queryAllLogBackupKeys failed: %v", err)
		}

		if state.CheckpointTS != 999999 {
			t.Errorf("expected CheckpointTS 999999, got %d", state.CheckpointTS)
		}

		if !state.InfoExists {
			t.Error("expected InfoExists to be true")
		}

		if state.IsPaused {
			t.Error("expected IsPaused to be false when pause key missing")
		}

		if state.KernelState != LogBackupKernelRunning {
			t.Errorf("expected KernelState running when not paused, got %s", state.KernelState)
		}
	})

	t.Run("checkpoint zero value", func(t *testing.T) {
		mockClient := &mockPDEtcdClient{
			getResponse: map[string][]*pdapi.KeyValue{
				"/tidb/br-stream/checkpoint/test-backup": {
					{Value: encodeUint64(0)},
				},
				"/tidb/br-stream/info/test-backup": {
					{Value: []byte("info")},
				},
			},
		}

		state, err := bt.queryAllLogBackupKeys(mockClient, "test-backup")
		if err != nil {
			t.Fatalf("queryAllLogBackupKeys failed: %v", err)
		}

		if state.CheckpointTS != 0 {
			t.Errorf("expected CheckpointTS 0, got %d", state.CheckpointTS)
		}
	})

	t.Run("checkpoint max value", func(t *testing.T) {
		maxUint64 := uint64(^uint64(0))
		mockClient := &mockPDEtcdClient{
			getResponse: map[string][]*pdapi.KeyValue{
				"/tidb/br-stream/checkpoint/test-backup": {
					{Value: encodeUint64(maxUint64)},
				},
			},
		}

		state, err := bt.queryAllLogBackupKeys(mockClient, "test-backup")
		if err != nil {
			t.Fatalf("queryAllLogBackupKeys failed: %v", err)
		}

		if state.CheckpointTS != maxUint64 {
			t.Errorf("expected CheckpointTS %d, got %d", maxUint64, state.CheckpointTS)
		}
	})

	t.Run("invalid checkpoint data", func(t *testing.T) {
		mockClient := &mockPDEtcdClient{
			getResponse: map[string][]*pdapi.KeyValue{
				"/tidb/br-stream/checkpoint/test-backup": {
					{Value: []byte("invalid")}, // Not 8 bytes
				},
			},
		}

		state, err := bt.queryAllLogBackupKeys(mockClient, "test-backup")
		if err != nil {
			t.Fatalf("queryAllLogBackupKeys failed: %v", err)
		}

		// Should handle gracefully, checkpoint stays 0
		if state.CheckpointTS != 0 {
			t.Errorf("expected CheckpointTS 0 for invalid data, got %d", state.CheckpointTS)
		}
	})

	t.Run("concurrent queries simulation", func(t *testing.T) {
		// Track query order to verify concurrency
		var queriedKeys []string
		var mu sync.Mutex

		mockClient := &mockPDEtcdClientWithTracking{
			getResponse: map[string][]*pdapi.KeyValue{
				"/tidb/br-stream/checkpoint/test-backup": {
					{Value: encodeUint64(123)},
				},
				"/tidb/br-stream/info/test-backup": {
					{Value: []byte("info")},
				},
				"/tidb/br-stream/pause/test-backup":      {},
				"/tidb/br-stream/last-error/test-backup": {},
			},
			onGet: func(key string) {
				mu.Lock()
				queriedKeys = append(queriedKeys, key)
				mu.Unlock()
				// Simulate some processing time
				time.Sleep(10 * time.Millisecond)
			},
		}

		state, err := bt.queryAllLogBackupKeys(mockClient, "test-backup")
		if err != nil {
			t.Fatalf("queryAllLogBackupKeys failed: %v", err)
		}

		// Verify all keys were queried
		if len(queriedKeys) != 4 {
			t.Errorf("expected 4 queries, got %d", len(queriedKeys))
		}

		// Verify state is correct despite concurrent queries
		if state.CheckpointTS != 123 {
			t.Errorf("expected CheckpointTS 123, got %d", state.CheckpointTS)
		}

		if !state.InfoExists {
			t.Error("expected InfoExists to be true")
		}
	})
}

func TestGetLogBackupKernelState(t *testing.T) {
	tests := []struct {
		paused   bool
		expected LogBackupKernelState
	}{
		{true, LogBackupKernelPaused},
		{false, LogBackupKernelRunning},
	}

	for _, tt := range tests {
		result := getLogBackupKernelState(tt.paused)
		if result != tt.expected {
			t.Errorf("getLogBackupKernelState(%v) = %s, want %s", tt.paused, result, tt.expected)
		}
	}
}

func TestIsCommandConsistentWithKernelState(t *testing.T) {
	tests := []struct {
		command     v1alpha1.LogSubCommandType
		kernelState LogBackupKernelState
		expected    bool
	}{
		{v1alpha1.LogStartCommand, LogBackupKernelRunning, true},
		{v1alpha1.LogResumeCommand, LogBackupKernelRunning, true},
		{v1alpha1.LogPauseCommand, LogBackupKernelPaused, true},
		{v1alpha1.LogStartCommand, LogBackupKernelPaused, false},
		{v1alpha1.LogPauseCommand, LogBackupKernelRunning, false},
	}

	for _, tt := range tests {
		result := isCommandConsistentWithKernelState(tt.command, tt.kernelState)
		if result != tt.expected {
			t.Errorf("isCommandConsistentWithKernelState(%s, %s) = %v, want %v",
				tt.command, tt.kernelState, result, tt.expected)
		}
	}
}

func TestParsePauseReason(t *testing.T) {
	bt := &backupTracker{}

	tests := []struct {
		name     string
		rawData  []byte
		expected string
	}{
		{
			name:     "V1 manual pause (empty data)",
			rawData:  []byte(""),
			expected: "manual",
		},
		{
			name: "V2 manual pause",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:         SeverityManual,
					OperatorHostName: "test-host",
					OperatorPID:      1234,
					OperationTime:    time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "manual",
		},
		{
			name: "V2 error pause with text/plain",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "text/plain",
					Payload:       []byte("Storage connection failed"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "Storage connection failed",
		},
		{
			name: "V2 error pause with text/plain charset",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "text/plain; charset=utf-8",
					Payload:       []byte("Disk full error"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "Disk full error",
		},
		{
			name: "V2 error pause with protobuf",
			rawData: func() []byte {
				// Create a StreamBackupError
				errorData := StreamBackupError{
					HappenAt:     uint64(time.Now().Unix() * 1000),
					ErrorCode:    "BR_001",
					ErrorMessage: "Failed to backup: disk full",
					StoreId:      5,
				}
				errorBytes, _ := json.Marshal(errorData)

				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "application/x-protobuf; messagetype=brpb.StreamBackupError",
					Payload:       errorBytes,
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "Paused by error(store 5): Failed to backup: disk full",
		},
		{
			name:     "V2 invalid JSON",
			rawData:  []byte("{invalid json}"),
			expected: "unknown",
		},
		{
			name: "V2 error with unsupported payload type",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "application/xml",
					Payload:       []byte("<error>test</error>"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "error",
		},
		{
			name: "V2 protobuf without messagetype",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "application/x-protobuf",
					Payload:       []byte("some data"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "error",
		},
		{
			name: "V2 protobuf with wrong messagetype",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "application/x-protobuf; messagetype=other.Type",
					Payload:       []byte("some data"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "error",
		},
		{
			name: "V2 protobuf with invalid error data",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "application/x-protobuf; messagetype=brpb.StreamBackupError",
					Payload:       []byte("invalid protobuf data"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "error",
		},
		{
			name: "V2 error with empty payload",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "text/plain",
					Payload:       []byte(""),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "",
		},
		{
			name: "V2 error with invalid mime type",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      SeverityError,
					PayloadType:   "not a valid mime type",
					Payload:       []byte("test"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "error",
		},
		{
			name: "V2 unknown severity",
			rawData: func() []byte {
				pauseInfo := PauseV2Info{
					Severity:      "UNKNOWN",
					PayloadType:   "text/plain",
					Payload:       []byte("test message"),
					OperationTime: time.Now(),
				}
				data, _ := json.Marshal(pauseInfo)
				return data
			}(),
			expected: "test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bt.parsePauseReason(tt.rawData)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

type mockPDEtcdClient struct {
	getResponse map[string][]*pdapi.KeyValue
	getError    error
}

func (m *mockPDEtcdClient) Get(key string, recursive bool) ([]*pdapi.KeyValue, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	if kvs, exists := m.getResponse[key]; exists {
		return kvs, nil
	}

	return []*pdapi.KeyValue{}, nil
}

func (m *mockPDEtcdClient) PutKey(key, value string) error {
	return nil
}

func (m *mockPDEtcdClient) PutTTLKey(key, value string, ttl int64) error {
	return nil
}

func (m *mockPDEtcdClient) DeleteKey(key string) error {
	return nil
}

func (m *mockPDEtcdClient) Close() error {
	return nil
}

// encodeUint64 encodes uint64 to big endian bytes
func encodeUint64(val uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	return buf
}

// mockPDEtcdClientWithTracking is a mock client that tracks Get calls
type mockPDEtcdClientWithTracking struct {
	getResponse map[string][]*pdapi.KeyValue
	getError    error
	onGet       func(key string)
}

func (m *mockPDEtcdClientWithTracking) Get(key string, recursive bool) ([]*pdapi.KeyValue, error) {
	if m.onGet != nil {
		m.onGet(key)
	}

	if m.getError != nil {
		return nil, m.getError
	}

	if kvs, exists := m.getResponse[key]; exists {
		return kvs, nil
	}

	return []*pdapi.KeyValue{}, nil
}

func (m *mockPDEtcdClientWithTracking) PutKey(key, value string) error {
	return nil
}

func (m *mockPDEtcdClientWithTracking) PutTTLKey(key, value string, ttl int64) error {
	return nil
}

func (m *mockPDEtcdClientWithTracking) DeleteKey(key string) error {
	return nil
}

func (m *mockPDEtcdClientWithTracking) Close() error {
	return nil
}

func TestDoubleCheckInconsistencyMechanism(t *testing.T) {
	bt := &backupTracker{
		logBackups: make(map[string]*trackDepends),
	}

	// Create a test backup
	backup := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: "default",
		},
		Spec: v1alpha1.BackupSpec{
			Mode:          v1alpha1.BackupModeLog,
			LogSubcommand: v1alpha1.LogPauseCommand,
		},
		Status: v1alpha1.BackupStatus{
			Phase: v1alpha1.BackupPaused, // Command completed, status updated
		},
	}

	// Add dependency to tracker with pre-cached state to avoid etcd calls
	logkey := genLogBackupKey("default", "test-backup")
	bt.logBackups[logkey] = &trackDepends{
		tc: &v1alpha1.TidbCluster{}, // Mock TC to avoid nil pointer
		state: &LogBackupState{
			InfoExists:    true,
			IsPaused:      false, // Kernel state is still Running (old cache)
			KernelState:   LogBackupKernelRunning,
			LastQueryTime: time.Now(), // Fresh cache to avoid refresh
		},
		lastRefresh:           time.Now(), // Set fresh refresh time
		inconsistencyDetected: false,
	}

	// Mock statusUpdater that tracks calls
	updateCalls := 0
	mockStatusUpdater := &mockStatusUpdater{
		updateFunc: func(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, status *controller.BackupUpdateStatus) error {
			updateCalls++
			return nil
		},
	}
	bt.statusUpdater = mockStatusUpdater

	// First call - should detect inconsistency but not correct
	canSkip, err := bt.SyncLogBackupState(backup)
	if err != nil {
		t.Fatalf("SyncLogBackupState failed: %v", err)
	}
	if !canSkip {
		t.Error("Expected to skip on first inconsistency detection")
	}
	if updateCalls != 0 {
		t.Errorf("Expected no status update on first detection, got %d calls", updateCalls)
	}

	// Check flag is set
	bt.operateLock.RLock()
	dep := bt.logBackups[logkey]
	bt.operateLock.RUnlock()
	if !dep.inconsistencyDetected {
		t.Error("Expected inconsistencyDetected to be true after first detection")
	}

	// Second call with same inconsistent state - should correct
	canSkip, err = bt.SyncLogBackupState(backup)
	if err != nil {
		t.Fatalf("SyncLogBackupState failed: %v", err)
	}
	if !canSkip {
		t.Error("Expected to skip after correction")
	}
	if updateCalls != 1 {
		t.Errorf("Expected 1 status update after correction, got %d calls", updateCalls)
	}

	// Check flag is cleared
	if dep.inconsistencyDetected {
		t.Error("Expected inconsistencyDetected to be false after correction")
	}
}

func TestDoubleCheckCacheDelayResolution(t *testing.T) {
	bt := &backupTracker{
		logBackups: make(map[string]*trackDepends),
	}

	backup := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: "default",
		},
		Spec: v1alpha1.BackupSpec{
			Mode:          v1alpha1.BackupModeLog,
			LogSubcommand: v1alpha1.LogPauseCommand,
		},
		Status: v1alpha1.BackupStatus{
			Phase: v1alpha1.BackupPaused, // Command completed, status updated
		},
	}

	// Add dependency with inconsistency flag already set
	logkey := genLogBackupKey("default", "test-backup")
	bt.logBackups[logkey] = &trackDepends{
		tc: &v1alpha1.TidbCluster{}, // Mock TC to avoid nil pointer
		state: &LogBackupState{
			InfoExists:    true,
			IsPaused:      true, // Cache refreshed, now consistent
			KernelState:   LogBackupKernelPaused,
			LastQueryTime: time.Now(), // Fresh cache to avoid refresh
		},
		lastRefresh:           time.Now(), // Set fresh refresh time
		inconsistencyDetected: true,       // Previously detected
	}

	updateCalls := 0
	mockStatusUpdater := &mockStatusUpdater{
		updateFunc: func(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, status *controller.BackupUpdateStatus) error {
			updateCalls++
			return nil
		},
	}
	bt.statusUpdater = mockStatusUpdater

	// Call with consistent state - should resolve and clear flag
	canSkip, err := bt.SyncLogBackupState(backup)
	if err != nil {
		t.Fatalf("SyncLogBackupState failed: %v", err)
	}
	if !canSkip {
		t.Error("Expected to skip when state is consistent")
	}
	if updateCalls != 0 {
		t.Errorf("Expected no status update when resolving cache delay, got %d calls", updateCalls)
	}

	// Check flag is cleared
	bt.operateLock.RLock()
	dep := bt.logBackups[logkey]
	bt.operateLock.RUnlock()
	if dep.inconsistencyDetected {
		t.Error("Expected inconsistencyDetected to be false after resolution")
	}
}

type mockStatusUpdater struct {
	updateFunc func(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, status *controller.BackupUpdateStatus) error
}

func (m *mockStatusUpdater) Update(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, status *controller.BackupUpdateStatus) error {
	if m.updateFunc != nil {
		return m.updateFunc(backup, condition, status)
	}
	return nil
}

func TestTryParseCheckpoint(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint64
	}{
		{
			name:     "valid 8-byte data",
			data:     encodeUint64(123456789),
			expected: 123456789,
		},
		{
			name:     "zero value",
			data:     encodeUint64(0),
			expected: 0,
		},
		{
			name:     "max uint64",
			data:     encodeUint64(^uint64(0)),
			expected: ^uint64(0),
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "too short (7 bytes)",
			data:     []byte{1, 2, 3, 4, 5, 6, 7},
			expected: 0,
		},
		{
			name:     "too long (9 bytes)",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
			expected: 0,
		},
		{
			name:     "invalid string data",
			data:     []byte("invalid"),
			expected: 0,
		},
		{
			name:     "nil data",
			data:     nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tryParseCheckpoint(tt.data)
			if result != tt.expected {
				t.Errorf("tryParseCheckpoint(%v) = %d, want %d", tt.data, result, tt.expected)
			}
		})
	}
}

func TestDoubleCheckCommandChangeReset(t *testing.T) {
	bt := &backupTracker{
		logBackups: make(map[string]*trackDepends),
	}

	// Create a test backup with initial pause command
	backup := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: "default",
		},
		Spec: v1alpha1.BackupSpec{
			Mode:          v1alpha1.BackupModeLog,
			LogSubcommand: v1alpha1.LogPauseCommand,
		},
	}

	// Add dependency with cache showing running state (inconsistent with pause)
	logkey := genLogBackupKey("default", "test-backup")
	bt.logBackups[logkey] = &trackDepends{
		tc: &v1alpha1.TidbCluster{},
		state: &LogBackupState{
			InfoExists:    true,
			IsPaused:      false,
			KernelState:   LogBackupKernelRunning,
			LastQueryTime: time.Now(),
		},
		lastRefresh:           time.Now(),
		inconsistencyDetected: false,
		lastCommand:           "", // No previous command
	}

	updateCalls := 0
	mockStatusUpdater := &mockStatusUpdater{
		updateFunc: func(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, status *controller.BackupUpdateStatus) error {
			updateCalls++
			return nil
		},
	}
	bt.statusUpdater = mockStatusUpdater

	// First call with pause command - should detect inconsistency
	canSkip, err := bt.SyncLogBackupState(backup)
	if err != nil {
		t.Fatalf("SyncLogBackupState failed: %v", err)
	}
	if !canSkip {
		t.Error("Expected to skip on first inconsistency detection")
	}
	if updateCalls != 0 {
		t.Errorf("Expected no status update on first detection, got %d calls", updateCalls)
	}

	// Check flag is set and command is recorded
	bt.operateLock.RLock()
	dep := bt.logBackups[logkey]
	bt.operateLock.RUnlock()
	if !dep.inconsistencyDetected {
		t.Error("Expected inconsistencyDetected to be true")
	}
	if dep.lastCommand != v1alpha1.LogPauseCommand {
		t.Errorf("Expected lastCommand to be %s, got %s", v1alpha1.LogPauseCommand, dep.lastCommand)
	}

	// Change command to resume (by setting start command but making backup appear paused)
	backup.Spec.LogSubcommand = v1alpha1.LogStartCommand
	backup.Status.Phase = v1alpha1.BackupPaused // This will make ParseLogBackupSubcommand return LogResumeCommand
	// Update state to show paused (still inconsistent with resume)
	dep.state = &LogBackupState{
		InfoExists:    true,
		IsPaused:      true,
		KernelState:   LogBackupKernelPaused,
		LastQueryTime: time.Now(),
	}
	dep.lastRefresh = time.Now()

	// Second call with resume command - should reset flag due to command change
	canSkip, err = bt.SyncLogBackupState(backup)
	if err != nil {
		t.Fatalf("SyncLogBackupState failed: %v", err)
	}
	if !canSkip {
		t.Error("Expected to skip on first inconsistency detection for new command")
	}
	if updateCalls != 0 {
		t.Errorf("Expected no status update on first detection of new command, got %d calls", updateCalls)
	}

	// Check that flag was reset and new command is recorded
	if !dep.inconsistencyDetected {
		t.Error("Expected inconsistencyDetected to be true for new command")
	}
	if dep.lastCommand != v1alpha1.LogResumeCommand {
		t.Errorf("Expected lastCommand to be %s, got %s", v1alpha1.LogResumeCommand, dep.lastCommand)
	}

	// Third call with same resume command - should now correct as second detection
	canSkip, err = bt.SyncLogBackupState(backup)
	if err != nil {
		t.Fatalf("SyncLogBackupState failed: %v", err)
	}
	if !canSkip {
		t.Error("Expected to skip after correction")
	}
	if updateCalls != 1 {
		t.Errorf("Expected 1 status update after second detection of same command, got %d calls", updateCalls)
	}

	// Check flag is cleared after correction
	if dep.inconsistencyDetected {
		t.Error("Expected inconsistencyDetected to be false after correction")
	}
}
