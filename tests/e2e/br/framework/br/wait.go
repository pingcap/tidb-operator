// Copyright 2021 PingCAP, Inc.
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
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	poll = time.Second * 2
)

// WaitForBackupDeleted will poll and wait until timeout or backup is really deleted
func WaitForBackupDeleted(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		if _, err := c.PingcapV1alpha1().Backups(ns).Get(name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			// check error is retriable
			return false, err
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for backup deleted: %v", err)
	}
	return nil
}

// WaitForBackupComplete will poll and wait until timeout or backup complete condition is true
func WaitForBackupComplete(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		b, err := c.PingcapV1alpha1().Backups(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range b.Status.Conditions {
			switch cond.Type {
			case v1alpha1.BackupComplete:
				if cond.Status == corev1.ConditionTrue {
					return true, nil
				}
			case v1alpha1.BackupFailed, v1alpha1.BackupInvalid:
				if cond.Status == corev1.ConditionTrue {
					return false, fmt.Errorf("backup is failed, reason: %s, message: %s", cond.Reason, cond.Message)
				}
			default: // do nothing
			}
		}

		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for backup complete: %v", err)
	}
	return nil
}

// WaitForRestoreComplete will poll and wait until timeout or restore complete condition is true
func WaitForRestoreComplete(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		r, err := c.PingcapV1alpha1().Restores(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range r.Status.Conditions {
			switch cond.Type {
			case v1alpha1.RestoreComplete:
				if cond.Status == corev1.ConditionTrue {
					return true, nil
				}
			case v1alpha1.RestoreFailed, v1alpha1.RestoreInvalid:
				if cond.Status == corev1.ConditionTrue {
					return false, fmt.Errorf("restore is failed, reason: %s, message: %s", cond.Reason, cond.Message)
				}
			default: // do nothing
			}
		}

		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for restore complete: %v", err)
	}
	return nil
}
