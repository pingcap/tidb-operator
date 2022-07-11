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

	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

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
	s, err := util.NewStorageBackend(backup.Spec.StorageProvider)
	if err != nil {
		return err
	}
	defer s.Close()

	opt := backup.GetCleanOption()

	klog.Infof("For backup %s, start to clean backup data with opt: %+v", bo, opt)

	index := 0
	count, deletedCount, failedCount := 0, 0, 0
	for {
		iter := s.ListPage(nil)
		objs, err := iter.Next(ctx, int(opt.PageSize))
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		for {
			// batch delete objects
			result := s.BatchDeleteObjects(ctx, objs, opt.BatchDeleteOption)

			index++
			count += len(objs)
			deletedCount += len(result.Deleted)
			failedCount += len(result.Errors)
			if len(result.Deleted) != 0 {
				klog.Infof("For backup %s, delete some objects successfully, index:%d deleted:%d", bo, index, len(result.Deleted))
				for _, obj := range result.Deleted {
					klog.V(4).Infof("For backup %s, delete object %s successfully", bo, obj)
				}
			}
			if len(result.Errors) != 0 {
				klog.Errorf("For backup %s, delete some objects failed, index:%d failed:%d", bo, index, len(result.Errors))
				for _, oerr := range result.Errors {
					klog.V(4).Infof("For backup %s, delete object %s failed: %s", bo, oerr.Key, oerr.Err)
				}
			}
			if len(result.Deleted)+len(result.Errors) < len(objs) {
				klog.Errorf("For backup %s, sum of deleted and failed is less than total, index:%d total:%d deleted:%d failed:%d",
					bo, len(objs), index, len(result.Deleted), len(result.Errors))
			}

			objs, err = iter.Next(ctx, int(opt.PageSize))
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
	}

	klog.Infof("For backup %s, clean backup finished, total:%d deleted:%d failed:%d", bo, count, deletedCount, failedCount)

	if deletedCount < count {
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
