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
	"strings"

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

	klog.Infof("cleanning cluster %s backup data with opt: %+v", bo, opt)

	iter := s.ListPage(nil)
	for {
		// list one page of object
		objs, err := iter.Next(ctx, int(opt.PageSize))
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		klog.Infof("For backup %s, try to delete %s objects", bo, len(objs))

		// batch delete objects
		result := s.BatchDeleteObjects(ctx, objs, opt.BatchDeleteOption)

		if len(result.Deleted) != 0 {
			klog.Infof("For backup %s, delete %s objects successfully", bo, len(result.Deleted))
			klog.V(4).Infof("For backup %s, all objects deleted: %s", bo, strings.Join(result.Deleted, ","))
		}
		if len(result.Deleted) < len(objs) {
			if len(result.Errors) != 0 {
				klog.Errorf("For backup %s, delete %s objects failed", bo, len(result.Errors))
				for _, oerr := range result.Errors {
					klog.V(4).Infof("For backup %s, delete object %s failed: %s", bo, oerr.Key, oerr.Err)
				}
			} else {
				klog.Errorf("For backup %s, deleted object is less than expected but failed is 0, deleted: %s, expected: %s", bo, len(result.Deleted), len(objs))
			}

			return fmt.Errorf("objects remain to delete")
		}
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
