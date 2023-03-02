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
	"context"
	"encoding/binary"
	"fmt"
	"path"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/tests/e2e/br/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

var (
	poll = time.Second * 2
)

// WaitForBackupDeleted will poll and wait until timeout or backup is really deleted
func WaitForBackupDeleted(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		if _, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{}); err != nil {
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
		b, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if b.Spec.Mode == v1alpha1.BackupModeLog {
			if v1alpha1.IsLogBackupSubCommandOntheCondition(b, v1alpha1.BackupComplete) {
				return true, nil
			}
			if v1alpha1.IsLogBackupSubCommandOntheCondition(b, v1alpha1.BackupFailed) || v1alpha1.IsLogBackupSubCommandOntheCondition(b, v1alpha1.BackupInvalid) {
				reason, message := v1alpha1.GetLogSubcommandConditionInfo(b)
				return false, fmt.Errorf("log backup is failed, reason: %s, message: %s", reason, message)
			}
		} else {
			for _, cond := range b.Status.Conditions {
				switch cond.Type {
				case v1alpha1.BackupComplete:
					if cond.Status == corev1.ConditionTrue {
						if cond.Status == corev1.ConditionTrue {
							return true, nil
						}
					}
				case v1alpha1.BackupFailed, v1alpha1.BackupInvalid:
					if cond.Status == corev1.ConditionTrue {
						return false, fmt.Errorf("backup is failed, reason: %s, message: %s", cond.Reason, cond.Message)
					}
				default: // do nothing
				}
			}
		}

		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for backup complete: %v", err)
	}
	return nil
}

// WaitForBackupOnRunning will poll and wait until timeout or backup phause is running
func WaitForBackupOnRunning(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		b, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if b.Status.Phase == v1alpha1.BackupRunning {
			return true, nil
		}

		for _, cond := range b.Status.Conditions {
			switch cond.Type {
			case v1alpha1.BackupFailed, v1alpha1.BackupInvalid:
				if cond.Status == corev1.ConditionTrue {
					return false, fmt.Errorf("backup is failed, reason: %s, message: %s", cond.Reason, cond.Message)
				}
			default: // do nothing
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for backup running: %v", err)
	}
	return nil
}

// WaitForBackupFailed will poll and wait until timeout or backup failed condition is true
func WaitForBackupFailed(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		b, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range b.Status.Conditions {
			switch cond.Type {
			case v1alpha1.BackupFailed:
				if cond.Status == corev1.ConditionTrue {
					if cond.Status == corev1.ConditionTrue {
						return true, nil
					}
				}
			case v1alpha1.BackupInvalid:
				if cond.Status == corev1.ConditionTrue {
					return false, fmt.Errorf("backup is invalid, reason: %s, message: %s", cond.Reason, cond.Message)
				}
			default: // do nothing
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for backup failed: %v", err)
	}
	return nil
}

// WaitForRestoreComplete will poll and wait until timeout or restore complete condition is true
func WaitForRestoreComplete(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		r, err := c.PingcapV1alpha1().Restores(ns).Get(context.TODO(), name, metav1.GetOptions{})
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

// WaitForLogBackupReachTS will poll and wait until timeout or log backup reach expect ts
func WaitForLogBackupReachTS(name, pdhost, expect string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll*5, timeout, func() (bool, error) {
		etcdCli, err := pdapi.NewPdEtcdClient(pdhost, 30*time.Second, nil)
		if err != nil {
			return false, err
		}
		streamKeyPrefix := "/tidb/br-stream"
		taskCheckpointPath := "/checkpoint"
		key := path.Join(streamKeyPrefix, taskCheckpointPath, name)
		kvs, err := etcdCli.Get(key, true)
		if err != nil {
			return false, err
		}
		if len(kvs) != 1 {
			return false, fmt.Errorf("get log backup checkpoint ts from pd %s failed", pdhost)
		}
		checkpointTS := binary.BigEndian.Uint64(kvs[0].Value)
		expectTS, err := config.ParseTSString(expect)

		if err != nil {
			return false, err
		}
		if checkpointTS >= expectTS {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for log backup reach ts complete: %v", err)
	}
	return nil
}

// WaitForRestoreProgressDone will poll and wait until timeout or restore progress has update to 100
func WaitForRestoreProgressDone(c versioned.Interface, ns, name string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll*5, timeout, func() (bool, error) {
		r, err := c.PingcapV1alpha1().Restores(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		count := len(r.Status.Progresses)
		if count == 0 {
			return false, nil
		}
		if r.Status.Progresses[count-1].Progress == 100 {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for restore progress done: %v", err)
	}
	return nil
}

// WaitForLogBackupProgressReachTS will poll and wait until timeout or log backup tracker has update checkpoint ts to expect
func WaitForLogBackupProgressReachTS(c versioned.Interface, ns, name, expect string, timeout time.Duration) error {
	if err := wait.PollImmediate(poll*5, timeout, func() (bool, error) {
		b, err := c.PingcapV1alpha1().Backups(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		expectTS, err := config.ParseTSString(expect)
		if err != nil {
			return false, err
		}
		checkpointTs, err := config.ParseTSString(b.Status.LogCheckpointTs)
		if err != nil {
			return false, err
		}
		if checkpointTs >= expectTS {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for log backup tracker reach ts complete: %v", err)
	}
	return nil
}

func WaitAndDeleteRunningBackupPod(f *framework.Framework, backup *v1alpha1.Backup, timeout time.Duration) error {
	ns := f.Namespace.Name
	name := backup.Name

	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		selector, err := label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name).Selector()
		if err != nil {
			return false, fmt.Errorf("fail to generate selector for backup %s/%s, error is %v", ns, name, err)
		}

		pods, err := f.ClientSet.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return false, fmt.Errorf("fail to list pods for backup %s/%s, error is %v", ns, name, err)
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				klog.Infof("skip delete not running pod %s, phase is %s", pod.Name, pod.Status.Phase)
				continue
			}
			err = f.ClientSet.CoreV1().Pods(ns).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
			if err != nil {
				return false, fmt.Errorf("fail to delete pod %s for backup %s/%s, error is %v", pod.Name, ns, name, err)
			}
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for delete running backup pod: %v", err)
	}
	return nil
}

func WaitBackupPodOnPhase(f *framework.Framework, backup *v1alpha1.Backup, phase corev1.PodPhase, timeout time.Duration) error {
	ns := f.Namespace.Name
	name := backup.Name

	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		selector, err := label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name).Selector()
		if err != nil {
			return false, fmt.Errorf("fail to generate selector for backup %s/%s, error is %v", ns, name, err)
		}

		pods, err := f.ClientSet.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return false, fmt.Errorf("fail to list pods for backup %s/%s, error is %v", ns, name, err)
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == phase {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for backup %s/%s pod on %s: %v", ns, name, phase, err)
	}
	return nil
}
