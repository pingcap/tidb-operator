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

package clean

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/cmd/tidb-backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/v2/cmd/tidb-backup-manager/app/util"
	bkutil "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
)

var defaultBackoff = wait.Backoff{
	Duration: 100 * time.Millisecond,
	Factor:   2.0,
	Jitter:   0,
	Steps:    8,
	Cap:      time.Second,
}

const (
	metaFile            = "/backupmeta"
	CloudAPIConcurrency = 3
)

// Options contains the input arguments to the backup command
type Options struct {
	Namespace  string
	BackupName string
}

func (bo *Options) String() string {
	return fmt.Sprintf("%s/%s", bo.Namespace, bo.BackupName)
}

// CleanBRRemoteBackupData clean the backup data from remote
func (bo *Options) CleanBRRemoteBackupData(ctx context.Context, backup *v1alpha1.Backup) error {
	opt := backup.GetCleanOption()

	backend, err := bkutil.NewStorageBackend(backup.Spec.StorageProvider, &bkutil.StorageCredential{})
	if err != nil {
		return err
	}
	defer backend.Close() //nolint:errcheck

	round := 0
	return util.RetryOnError(ctx, opt.RetryCount, 0, util.RetriableOnAnyError, func() error {
		round++
		err := bo.cleanBRRemoteBackupDataOnce(ctx, backend, opt, round)
		if err != nil {
			klog.Errorf("For backup %s clean %d, failed to clean backup: %s", bo, round, err)
		}
		return err
	})
}

func (bo *Options) cleanBRRemoteBackupDataOnce(ctx context.Context,
	backend *bkutil.StorageBackend,
	opt v1alpha1.CleanOption,
	round int,
) error {
	klog.Infof("For backup %s clean %d, start to clean backup with opt: %+v", bo, round, opt)

	iter := backend.ListPage(nil)
	backoff := defaultBackoff
	index := 0
	count, deletedCount, failedCount := 0, 0, 0
	for {
		needBackoff := false
		index++
		logPrefix := fmt.Sprintf("For backup %s clean %d-%d", bo, round, index)

		objs, err := iter.Next(ctx, int(opt.PageSize)) //nolint:gosec
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		klog.Infof("%s, try to delete %d objects", logPrefix, len(objs))
		result := backend.BatchDeleteObjects(ctx, objs, opt.BatchDeleteOption)

		count += len(objs)
		deletedCount += len(result.Deleted)
		failedCount += len(result.Errors)

		if len(result.Deleted) != 0 {
			klog.Infof("%s, delete %d objects successfully", logPrefix, len(result.Deleted))
			for _, obj := range result.Deleted {
				klog.V(4).Infof("%s, delete object %s successfully", logPrefix, obj)
			}
		}
		if len(result.Errors) != 0 {
			klog.Errorf("%s, delete %d objects failed", logPrefix, len(result.Errors))
			for _, oerr := range result.Errors {
				klog.V(4).Infof("%s, delete object %s failed: %s", logPrefix, oerr.Key, oerr.Err)
			}
			needBackoff = true
		}
		if len(result.Deleted)+len(result.Errors) < len(objs) {
			klog.Errorf("%s, sum of deleted and failed objects %d is less than expected", logPrefix, len(result.Deleted)+len(result.Errors))
			needBackoff = true
		}

		if opt.BackoffEnabled {
			if needBackoff {
				time.Sleep(backoff.Step())
			} else {
				backoff = defaultBackoff // reset backoff
			}
		}
	}

	klog.Infof("For backup %s clean %d, clean backup finished, total:%d deleted:%d failed:%d", bo, round, count, deletedCount, failedCount)

	if deletedCount < count {
		return fmt.Errorf("some objects failed to be deleted")
	}

	objs, err := backend.ListPage(nil).Next(ctx, int(opt.PageSize)) //nolint:gosec
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(objs) != 0 {
		return fmt.Errorf("some objects are missing to be deleted")
	}

	return nil
}

func (bo *Options) cleanRemoteBackupData(ctx context.Context, bucket string, opts []string) error {
	destBucket := util.NormalizeBucketURI(bucket)
	args := util.ConstructRcloneArgs(constants.RcloneConfigArg, opts, "delete", destBucket, "", true)
	output, err := exec.CommandContext(ctx, "rclone", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone delete command failed, output: %s, err: %w", bo, string(output), err)
	}

	args = util.ConstructRcloneArgs(constants.RcloneConfigArg, opts, "delete", fmt.Sprintf("%s.tmp", destBucket), "", true)
	output, err = exec.CommandContext(ctx, "rclone", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone delete command failed, output: %s, err: %w", bo, string(output), err)
	}

	klog.Infof("cluster %s backup %s was deleted successfully", bo, bucket)
	return nil
}
