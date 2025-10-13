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

package kernelSyncUtil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Network failure simulation functions

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

// Backup verification functions

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

// WaitForBackupErrorPause waits for backup to enter RetryTheFailed state due to network error
func WaitForBackupErrorPause(c versioned.Interface, ns, backupName string) error {
	return wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
		currentBackup, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), backupName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check for RetryTheFailed condition with error pause
		_, condition := v1alpha1.GetBackupCondition(&currentBackup.Status, v1alpha1.BackupRetryTheFailed)
		if condition == nil || condition.Status != v1.ConditionTrue {
			return false, nil // Keep waiting, no error
		}

		if condition.Reason != "LogBackupErrorPaused" {
			return false, fmt.Errorf("expected reason LogBackupErrorPaused, got %s", condition.Reason)
		}

		// Verify error message contains network-related keywords
		if !strings.Contains(condition.Message, "connection") &&
			!strings.Contains(condition.Message, "network") &&
			!strings.Contains(condition.Message, "timeout") {
			return false, fmt.Errorf("error message doesn't contain network keywords: %s", condition.Message)
		}

		log.Logf("Detected error pause: %s", condition.Message)
		return true, nil
	})
}

// WaitForBackupRecovery waits for backup to recover from error state to Running state
func WaitForBackupRecovery(c versioned.Interface, ns, backupName string) error {
	return wait.PollImmediate(10*time.Second, 3*time.Minute, func() (bool, error) {
		currentBackup, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), backupName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check that RetryTheFailed condition is cleared or false
		_, condition := v1alpha1.GetBackupCondition(&currentBackup.Status, v1alpha1.BackupRetryTheFailed)
		if condition != nil && condition.Status == v1.ConditionTrue {
			return false, nil // Keep waiting, no error
		}

		// Verify backup is running again
		if currentBackup.Status.Phase != v1alpha1.BackupRunning {
			return false, nil // Keep waiting, no error
		}

		log.Logf("Backup recovered to Running state")
		return true, nil
	})
}
