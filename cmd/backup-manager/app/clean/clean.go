// Copyright 2019 PingCAP, Inc.
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

package clean

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

const (
	retryCount = 5
)

type stat struct {
	index        int
	totalCount   int
	deletedCount int
	failedCount  int
}

// Options contains the input arguments to the backup command
type Options struct {
	Namespace  string
	BackupName string
}

func (bo *Options) String() string {
	return fmt.Sprintf("%s/%s", bo.Namespace, bo.BackupName)
}

// cleanBRRemoteBackupData clean the backup data from remote
func (bo *Options) cleanBRRemoteBackupData(ctx context.Context, backup *v1alpha1.Backup) error {

	stat := &stat{
		index: 1,
	}

	for stat.index < retryCount {
		err := bo.cleanBRRemoteBackupDataOnce(ctx, backup, stat)
		if err == nil {
			return nil
		}
		stat.index++
		// log
	}

	return fmt.Errorf("some objects still remain to delete")
}

func (bo *Options) cleanBRRemoteBackupDataOnce(ctx context.Context, backup *v1alpha1.Backup, stat *stat) error {
	logPrefix := fmt.Sprintf("For backup %s clean %d", bo, stat.index)

	s, err := util.NewStorageBackend(backup.Spec.StorageProvider)
	if err != nil {
		return err
	}
	defer s.Close()

	iter := s.ListPage(nil)
	opt := backup.GetCleanOption()

	backoff := wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0,
		Steps:    8,
		Cap:      time.Second,
	}
	count, deletedCount, failedCount := 0, 0, 0

	backoffFor(backoff, func() (bool, bool, error) {
		objs, err := iter.Next(ctx, int(opt.PageSize))
		if err == io.EOF {
			return true, false, nil
		}
		if err != nil {
			return false, false, err
		}

		// batch delete objects
		result := s.BatchDeleteObjects(ctx, objs, opt.BatchDeleteOption)

		count += len(objs)
		deletedCount += len(result.Deleted)
		failedCount += len(result.Errors)
		needBackOff := false

		if len(result.Deleted) != 0 {
			klog.Infof("%s, delete some objects successfully, deleted:%d", logPrefix, len(result.Deleted))
			for _, obj := range result.Deleted {
				klog.V(4).Infof("%s, delete object %s successfully", logPrefix, obj)
			}
		}
		if len(result.Errors) != 0 {
			klog.Errorf("%s, delete some objects failed, failed:%d", logPrefix, len(result.Errors))
			for _, oerr := range result.Errors {
				klog.V(4).Infof("%s, delete object %s failed: %s", logPrefix, oerr.Key, oerr.Err)
			}
			needBackOff = true
		}
		if len(result.Deleted)+len(result.Errors) < len(objs) {
			klog.Errorf("%s, sum of deleted and failed is less than total,total:%d deleted:%d failed:%d",
				logPrefix, len(objs), len(result.Deleted), len(result.Errors))
			needBackOff = true
		}

		return false, needBackOff, nil
	})

	stat.totalCount += count
	stat.deletedCount += deletedCount
	stat.failedCount += failedCount

	if deletedCount < count {
		return fmt.Errorf("some objects remain to delete")
	}

	// double check
	iter = s.ListPage(nil)
	objs, err := iter.Next(ctx, int(opt.PageSize))
	if err != nil && err != io.EOF {
		return err
	}
	if len(objs) != 0 {
		return fmt.Errorf("some objects remain to delete")
	}

	return nil
}

func (bo *Options) cleanRemoteBackupData(ctx context.Context, bucket string, opts []string) error {
	destBucket := util.NormalizeBucketURI(bucket)
	args := util.ConstructRcloneArgs(constants.RcloneConfigArg, opts, "delete", destBucket, "", true)
	output, err := exec.CommandContext(ctx, "rclone", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone delete command failed, output: %s, err: %v", bo, string(output), err)
	}

	args = util.ConstructRcloneArgs(constants.RcloneConfigArg, opts, "delete", fmt.Sprintf("%s.tmp", destBucket), "", true)
	output, err = exec.CommandContext(ctx, "rclone", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone delete command failed, output: %s, err: %v", bo, string(output), err)
	}

	klog.Infof("cluster %s backup %s was deleted successfully", bo, bucket)
	return nil
}

func backoffFor(oriBackoff wait.Backoff, f func() (bool /*done*/, bool /*backoff*/, error)) error {
	usedBackoff := oriBackoff

	for {
		done, backoff, err := f()
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		if !backoff {
			usedBackoff = oriBackoff
			continue
		}

		time.Sleep(usedBackoff.Step())
	}
}
