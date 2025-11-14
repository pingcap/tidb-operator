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

package framework

import (
	"context"
	"encoding/binary"
	"fmt"
	"path"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
)

var (
	poll = time.Second * 2
)

func GetAndCheckBackup(c client.Client, ns, name string, check func(b *v1alpha1.Backup) bool) error {
	b := &v1alpha1.Backup{}
	if err := c.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: name}, b); err != nil {
		return fmt.Errorf("can't get backup %s/%s: %w", ns, name, err)
	}
	if !check(b) {
		return fmt.Errorf("backup %s/%s is not in expected state", ns, name)
	}
	return nil
}

// WaitForBackup will poll and wait until timeout
func WaitForBackup(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		b := &v1alpha1.Backup{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, b); err != nil {
			return false, err
		}
		for _, cond := range b.Status.Conditions {
			switch v1alpha1.BackupConditionType(cond.Type) {
			case v1alpha1.BackupFailed:
				if cond.Status == metav1.ConditionTrue {
					if cond.Status == metav1.ConditionTrue {
						return true, nil
					}
				}
			case v1alpha1.BackupInvalid:
				if cond.Status == metav1.ConditionTrue {
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

// WaitForBackupDeleted will poll and wait until timeout or backup is really deleted
func WaitForBackupDeleted(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &v1alpha1.Backup{}); err != nil {
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

// WaitForRestoreDeleted will poll and wait until timeout or restore is really deleted
func WaitForRestoreDeleted(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &v1alpha1.Restore{}); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			// check error is retriable
			return false, err
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for restore deleted: %v", err)
	}
	return nil
}

// WaitForBackupComplete will poll and wait until timeout or backup complete condition is true
func WaitForBackupComplete(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		b := &v1alpha1.Backup{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, b); err != nil {
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
				switch v1alpha1.BackupConditionType(cond.Type) {
				case v1alpha1.BackupComplete:
					if cond.Status == metav1.ConditionTrue {
						if cond.Status == metav1.ConditionTrue {
							return true, nil
						}
					}
				case v1alpha1.BackupFailed, v1alpha1.BackupInvalid:
					if cond.Status == metav1.ConditionTrue {
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
func WaitForBackupOnRunning(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		b := &v1alpha1.Backup{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, b); err != nil {
			return false, err
		}
		if b.Status.Phase == v1alpha1.BackupRunning {
			return true, nil
		}

		for _, cond := range b.Status.Conditions {
			switch v1alpha1.BackupConditionType(cond.Type) {
			case v1alpha1.BackupFailed, v1alpha1.BackupInvalid:
				if cond.Status == metav1.ConditionTrue {
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

// WaitForBackupOnRunning will poll and wait until timeout or backup phause is schedule
func WaitForBackupOnScheduled(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		b := &v1alpha1.Backup{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, b); err != nil {
			return false, err
		}
		if b.Status.Phase == v1alpha1.BackupScheduled {
			return true, nil
		}

		for _, cond := range b.Status.Conditions {
			switch v1alpha1.BackupConditionType(cond.Type) {
			case v1alpha1.BackupFailed, v1alpha1.BackupInvalid:
				if cond.Status == metav1.ConditionTrue {
					return false, fmt.Errorf("backup is failed, reason: %s, message: %s", cond.Reason, cond.Message)
				}
			default: // do nothing
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("can't wait for backup scheduled: %v", err)
	}
	return nil
}

// WaitForBackupFailed will poll and wait until timeout or backup failed condition is true
func WaitForBackupFailed(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		b := &v1alpha1.Backup{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, b); err != nil {
			return false, err
		}
		for _, cond := range b.Status.Conditions {
			switch v1alpha1.BackupConditionType(cond.Type) {
			case v1alpha1.BackupFailed:
				if cond.Status == metav1.ConditionTrue {
					if cond.Status == metav1.ConditionTrue {
						return true, nil
					}
				}
			case v1alpha1.BackupInvalid:
				if cond.Status == metav1.ConditionTrue {
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
func WaitForRestoreComplete(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		r := &v1alpha1.Restore{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, r); err != nil {
			return false, err
		}
		for _, cond := range r.Status.Conditions {
			switch v1alpha1.RestoreConditionType(cond.Type) {
			case v1alpha1.RestoreComplete:
				if cond.Status == metav1.ConditionTrue {
					return true, nil
				}
			case v1alpha1.RestoreFailed, v1alpha1.RestoreInvalid:
				if cond.Status == metav1.ConditionTrue {
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
		if len(kvs) == 0 {
			// wait for log backup start
			return false, nil
		}
		if len(kvs) > 1 {
			return false, fmt.Errorf("get log backup checkpoint ts from pd %s failed, expect 1, got %d", pdhost, len(kvs))
		}
		checkpointTS := binary.BigEndian.Uint64(kvs[0].Value)
		expectTS, err := v1alpha1.ParseTSString(expect)

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
func WaitForRestoreProgressDone(c client.Client, ns, name string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		r := &v1alpha1.Restore{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, r); err != nil {
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
func WaitForLogBackupProgressReachTS(c client.Client, ns, name, expect string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.TODO(), poll, timeout, true, func(ctx context.Context) (bool, error) {
		b := &v1alpha1.Backup{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, b); err != nil {
			return false, err
		}
		expectTS, err := v1alpha1.ParseTSString(expect)
		if err != nil {
			return false, err
		}
		checkpointTs, err := v1alpha1.ParseTSString(b.Status.LogCheckpointTs)
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

func WaitAndDeleteRunningBackupPod(f *Framework, backup *v1alpha1.Backup, timeout time.Duration) error {
	ns := f.Namespace.Name
	name := backup.Name

	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		selector, err := metav1alpha1.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name).Selector()
		if err != nil {
			return false, fmt.Errorf("fail to generate selector for backup %s/%s, error is %v", ns, name, err)
		}

		var pods corev1.PodList
		err = f.Client.List(context.TODO(), &pods, client.InNamespace(ns), &client.ListOptions{LabelSelector: selector})
		if err != nil {
			return false, fmt.Errorf("fail to list pods for backup %s/%s, error is %v", ns, name, err)
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}
			err = f.Client.Delete(context.TODO(), &pod)
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

func WaitBackupPodOnPhase(f *Framework, backup *v1alpha1.Backup, phase corev1.PodPhase, timeout time.Duration) error {
	ns := f.Namespace.Name
	name := backup.Name

	if err := wait.PollImmediate(poll, timeout, func() (bool, error) {
		selector, err := metav1alpha1.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name).Selector()
		if err != nil {
			return false, fmt.Errorf("fail to generate selector for backup %s/%s, error is %v", ns, name, err)
		}

		var pods corev1.PodList
		err = f.Client.List(context.TODO(), &pods, client.InNamespace(ns), &client.ListOptions{LabelSelector: selector})
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

// func printPodLogs(f *framework.Framework, ns, name string) {
// 	var pods corev1.PodList
// 	err := f.Client.List(context.TODO(), &pods, &client.ListOptions{})
// 	if err != nil {
// 		log.Logf("Error listing pods: %v", err)
// 		return
// 	}

// 	var matchingPods []v1.Pod
// 	for _, pod := range pods.Items {
// 		if strings.Contains(pod.Name, name) {
// 			matchingPods = append(matchingPods, pod)
// 		}
// 	}

// 	if len(matchingPods) == 0 {
// 		fmt.Printf("No pods found containing '%s' in namespace '%s'\n", name, ns)
// 		return
// 	}

// 	for _, pod := range matchingPods {
// 		req := f.ClientSet.CoreV1().Pods(ns).GetLogs(pod.Name, &v1.PodLogOptions{})

// 		// Execute the log request and get the stream
// 		logStream, err := req.Stream(context.TODO())
// 		if err != nil {
// 			log.Logf("Error retrieving logs for pod %s: %v", pod.Name, err)
// 			return
// 		}
// 		defer logStream.Close()

// 		// Read the log stream
// 		logBytes, err := io.ReadAll(logStream)
// 		if err != nil {
// 			log.Logf("Error reading logs for pod %s: %v", pod.Name, err)
// 			return
// 		}

// 		// Print the logs as a string
// 		log.Logf("Logs for pod %s in namespace %s: %s", pod.Name, ns, string(logBytes))
// 	}
// }
