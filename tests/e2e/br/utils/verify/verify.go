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

package verify

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/types"
	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// VerifyOperatorDetectsInconsistency checks that operator detects kernel-operator mismatch
func VerifyOperatorDetectsInconsistency(c versioned.Interface, ns, backupName string, expectedCondition v1alpha1.BackupConditionType) error {
	return wait.PollImmediate(10*time.Second, 3*time.Minute, func() (bool, error) {
		backup, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), backupName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		
		// Look for the expected condition
		for _, condition := range backup.Status.Conditions {
			if condition.Type == expectedCondition && condition.Status == corev1.ConditionTrue {
				log.Logf("Found expected condition %s: %s", expectedCondition, condition.Message)
				return true, nil
			}
		}
		
		log.Logf("Waiting for condition %s, current conditions: %+v", expectedCondition, backup.Status.Conditions)
		return false, nil
	})
}

// VerifyKernelState directly queries etcd to confirm actual kernel state
func VerifyKernelState(etcdClient pdapi.PDEtcdClient, backupName string, expectedState types.LogBackupKernelState) error {
	// Query all log backup keys to get complete state
	state, err := QueryLogBackupStateFromEtcd(etcdClient, backupName)
	if err != nil {
		return err
	}
	
	if state.KernelState != expectedState {
		return fmt.Errorf("expected kernel state %s, got %s (IsPaused=%v, PauseReason=%s)", 
			expectedState, state.KernelState, state.IsPaused, state.PauseReason)
	}
	
	log.Logf("Verified kernel state: %s (IsPaused=%v, Reason=%s)", 
		state.KernelState, state.IsPaused, state.PauseReason)
	return nil
}

// QueryLogBackupStateFromEtcd directly queries etcd for complete log backup state
func QueryLogBackupStateFromEtcd(etcdClient pdapi.PDEtcdClient, backupName string) (*types.LogBackupState, error) {
	state := &types.LogBackupState{
		LastQueryTime: time.Now(),
	}
	
	// Query checkpoint
	checkpointKey := fmt.Sprintf("/tidb/br-stream/checkpoint/%s", backupName)
	kvs, err := etcdClient.Get(checkpointKey, false)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoint: %v", err)
	}
	if len(kvs) > 0 && len(kvs[0].Value) == 8 {
		state.CheckpointTS = binary.BigEndian.Uint64(kvs[0].Value)
	}
	
	// Query info key
	infoKey := fmt.Sprintf("/tidb/br-stream/info/%s", backupName)
	kvs, err = etcdClient.Get(infoKey, false)
	if err != nil {
		return nil, fmt.Errorf("failed to query info: %v", err)
	}
	state.InfoExists = len(kvs) > 0
	
	// Query pause key
	pauseKey := fmt.Sprintf("/tidb/br-stream/pause/%s", backupName)
	kvs, err = etcdClient.Get(pauseKey, false)
	if err != nil {
		return nil, fmt.Errorf("failed to query pause: %v", err)
	}
	
	if len(kvs) > 0 {
		state.IsPaused = true
		pauseData := kvs[0].Value
		
		// Parse pause reason
		if len(pauseData) == 0 {
			// V1 format: empty = manual
			state.PauseReason = "manual"
		} else {
			// Try V2 format
			var pauseInfo struct {
				Severity    string `json:"severity"`
				PayloadType string `json:"payload_type"`
				Payload     []byte `json:"payload"`
			}
			
			if err := json.Unmarshal(pauseData, &pauseInfo); err == nil {
				if pauseInfo.Severity == "MANUAL" {
					state.PauseReason = "manual"
				} else if pauseInfo.Severity == "ERROR" {
					// Parse payload based on type
					if strings.HasPrefix(pauseInfo.PayloadType, "text/plain") {
						state.PauseReason = string(pauseInfo.Payload)
					} else if strings.Contains(pauseInfo.PayloadType, "protobuf") {
						// Parse protobuf error
						var errorData struct {
							ErrorMessage string `json:"error_message"`
							StoreId      uint64 `json:"store_id"`
						}
						if err := json.Unmarshal(pauseInfo.Payload, &errorData); err == nil {
							state.PauseReason = fmt.Sprintf("Paused by error(store %d): %s", 
								errorData.StoreId, errorData.ErrorMessage)
						} else {
							state.PauseReason = "error"
						}
					} else {
						state.PauseReason = "error"
					}
				} else {
					state.PauseReason = "unknown"
				}
			} else {
				// Fallback to raw data
				state.PauseReason = string(pauseData)
			}
		}
	}
	
	// Determine kernel state
	if state.IsPaused {
		state.KernelState = types.LogBackupKernelPaused
	} else {
		state.KernelState = types.LogBackupKernelRunning
	}
	
	// Query error key
	errorKey := fmt.Sprintf("/tidb/br-stream/last-error/%s", backupName)
	kvs, err = etcdClient.Get(errorKey, false)
	if err != nil {
		return nil, fmt.Errorf("failed to query error: %v", err)
	}
	if len(kvs) > 0 {
		state.LastError = string(kvs[0].Value)
	}
	
	return state, nil
}

// VerifyTimeSyncedUpdated checks that TimeSynced field is updated after sync
func VerifyTimeSyncedUpdated(c versioned.Interface, ns, backupName string, beforeTime time.Time) error {
	return wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		backup, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), backupName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		
		if backup.Status.TimeSynced == nil {
			log.Logf("TimeSynced field is nil, waiting...")
			return false, nil
		}
		
		if backup.Status.TimeSynced.After(beforeTime) {
			log.Logf("TimeSynced updated: %v (after %v)", backup.Status.TimeSynced.Time, beforeTime)
			return true, nil
		}
		
		log.Logf("TimeSynced not updated yet: %v (before %v)", backup.Status.TimeSynced.Time, beforeTime)
		return false, nil
	})
}

// VerifyCheckpointProgression checks that checkpoint timestamp is advancing
func VerifyCheckpointProgression(etcdClient pdapi.PDEtcdClient, backupName string, initialCheckpoint uint64) error {
	return wait.PollImmediate(30*time.Second, 5*time.Minute, func() (bool, error) {
		state, err := QueryLogBackupStateFromEtcd(etcdClient, backupName)
		if err != nil {
			return false, err
		}
		
		if state.CheckpointTS > initialCheckpoint {
			log.Logf("Checkpoint progressed: %d -> %d", initialCheckpoint, state.CheckpointTS)
			return true, nil
		}
		
		log.Logf("Checkpoint not progressed yet: %d (initial: %d)", state.CheckpointTS, initialCheckpoint)
		return false, nil
	})
}

// WaitForKernelStateChange polls etcd until kernel state matches expected state
func WaitForKernelStateChange(etcdClient pdapi.PDEtcdClient, backupName string, expectedState types.LogBackupKernelState, timeout time.Duration) error {
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		state, err := QueryLogBackupStateFromEtcd(etcdClient, backupName)
		if err != nil {
			log.Logf("Failed to query kernel state: %v", err)
			return false, nil // Continue polling
		}
		
		if state.KernelState == expectedState {
			log.Logf("Kernel state changed to %s as expected", expectedState)
			return true, nil
		}
		
		log.Logf("Waiting for kernel state to change to %s, current: %s", expectedState, state.KernelState)
		return false, nil
	})
}