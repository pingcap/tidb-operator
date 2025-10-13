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
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import "time"

// LogBackupState mirrors the structure from backup_tracker.go
type LogBackupState struct {
	CheckpointTS  uint64
	IsPaused      bool
	PauseReason   string
	LastError     string
	InfoExists    bool
	KernelState   LogBackupKernelState
	LastQueryTime time.Time
	QueryError    error
}

type LogBackupKernelState string

const (
	LogBackupKernelRunning LogBackupKernelState = "running"
	LogBackupKernelPaused  LogBackupKernelState = "paused"
)