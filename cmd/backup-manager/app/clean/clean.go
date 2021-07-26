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

	"k8s.io/klog"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

const (
	pageSize             = 10000
	pageConcurrency      = 10
	goroutineConcurrency = 100
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

	iter := util.ListPage(s, nil)
	for {
		// list one page of object
		objs, err := iter.Next(ctx, pageSize)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// batch delete objects
		var s3cli *s3.S3
		var result *util.BatchDeleteObjectsResult
		if ok := s.As(&s3cli); ok {
			klog.Infof("Delete a page of object by S3 batch delete")
			bucket := backup.Spec.StorageProvider.S3.Bucket
			prefix := backup.Spec.StorageProvider.S3.Prefix
			for i := range objs {
				objs[i].Key = fmt.Sprintf("%s/%s", prefix, objs[i].Key)
			}
			result = util.BatchDeleteObjectsOfS3(ctx, s3cli, bucket, objs, pageConcurrency)
		} else {
			klog.Infof("Delete a page of object concurrently")
			result = util.BatchDeleteObjectsConcurrently(ctx, s, objs, goroutineConcurrency)
		}

		if len(result.Deleted) != 0 {
			klog.Infof("Delete these objects for cluster successfully: %s", strings.Join(result.Deleted, ","))
		}
		if len(result.Errors) != 0 {
			for _, oerr := range result.Errors {
				klog.Errorf("Delete object %s failed: %s", oerr.Key, oerr.Err)
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
