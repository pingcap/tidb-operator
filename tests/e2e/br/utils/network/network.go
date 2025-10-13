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

package network

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/types"
	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SimulateMinIONetworkFailure creates MinIO connection failure that triggers kernel auto-pause
func SimulateMinIONetworkFailure(clientset kubernetes.Interface, ns, clusterName string) error {
	// Network policy to block MinIO access (E2E uses MinIO, not real S3)
	blockMinIOPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "block-minio-access",
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "tikv",
					"app.kubernetes.io/instance":  clusterName,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// Allow namespace-internal communication (PD, etc) but exclude MinIO
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": ns,
								},
							},
						},
					},
				},
				{
					// Allow external communication except MinIO service
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	}

	_, err := clientset.NetworkingV1().NetworkPolicies(ns).Create(context.TODO(), blockMinIOPolicy, metav1.CreateOptions{})
	return err
}

// RestoreMinIONetworkAccess restores normal MinIO connectivity
func RestoreMinIONetworkAccess(clientset kubernetes.Interface, ns, clusterName string) error {
	// Remove network policy to restore MinIO access
	err := clientset.NetworkingV1().NetworkPolicies(ns).Delete(
		context.TODO(), "block-minio-access", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	log.Logf("MinIO network access restored for cluster %s", clusterName)
	return nil
}

// WaitForKernelAutoPause waits for the log backup kernel to enter paused state.
//
// Core Purpose:
//   - Used in network failure tests to verify that TiDB's log backup kernel
//     automatically pauses when it cannot write to storage (S3/MinIO)
//   - Polls etcd's /tidb/br-stream/pause/{backupName} key until it appears
//   - Returns the pause reason for test verification
//
// Context:
//   - Called after SimulateMinIONetworkFailure() blocks storage access
//   - Tests expect kernel to detect storage failure and auto-pause within timeout
func WaitForKernelAutoPause(etcdClient pdapi.PDEtcdClient, backupName string, timeout time.Duration) (*types.LogBackupState, error) {
	log.Logf("Waiting for kernel auto-pause of backup '%s' within %v", backupName, timeout)
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		// Check if pause key exists in etcd
		pauseKey := fmt.Sprintf("/tidb/br-stream/pause/%s", backupName)
		kvs, err := etcdClient.Get(pauseKey, false)
		if err != nil {
			return nil, fmt.Errorf("failed to query etcd pause key: %v", err)
		}

		// If pause key exists, kernel has paused the backup
		if len(kvs) > 0 {
			pauseData := kvs[0].Value
			pauseReason := "unknown"

			// Try to extract pause reason from the data
			if len(pauseData) > 0 {
				var pauseInfo struct {
					Severity string `json:"severity"`
					Payload  []byte `json:"payload"`
				}
				if err := json.Unmarshal(pauseData, &pauseInfo); err == nil {
					pauseReason = string(pauseInfo.Payload)
				} else {
					// Fallback: use raw data as reason
					pauseReason = string(pauseData)
				}
			}

			log.Logf("Kernel auto-pause detected after %v, reason: %s", time.Since(startTime), pauseReason)
			return &types.LogBackupState{
				IsPaused:      true,
				PauseReason:   pauseReason,
				KernelState:   types.LogBackupKernelPaused,
				LastQueryTime: time.Now(),
			}, nil
		}

		log.Logf("Pause key not found yet, continuing to wait... (elapsed: %v)", time.Since(startTime))
		time.Sleep(10 * time.Second)
	}

	return nil, fmt.Errorf("timeout waiting for kernel auto-pause after %v", timeout)
}
