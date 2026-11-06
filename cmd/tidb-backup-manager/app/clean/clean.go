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
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
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
	if err := validateBRBackupPath(backup); err != nil {
		return err
	}
	if _, err := bkutil.ValidateStorageIdentity(backup); err != nil {
		return err
	}
	if bkutil.GetStorageType(backup.Spec.StorageProvider) == v1alpha1.BackupStorageTypeLocal {
		return cleanLocalBackupData(ctx, backup)
	}
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

// Validate before opening storage: the live spec must still identify the recorded backup.
func validateBRBackupPath(backup *v1alpha1.Backup) error {
	target, err := util.GetStoragePath(&backup.Spec.StorageProvider)
	if err != nil {
		return err
	}
	if backup.Status.BackupPath == "" || target != backup.Status.BackupPath {
		return fmt.Errorf("refusing cleanup: storage target does not match status.backupPath")
	}
	var bucket, prefix, scheme, suffix string
	switch bkutil.GetStorageType(backup.Spec.StorageProvider) {
	case v1alpha1.BackupStorageTypeS3:
		bucket, prefix, scheme = backup.Spec.S3.Bucket, backup.Spec.S3.Prefix, "s3"
	case v1alpha1.BackupStorageTypeGcs:
		bucket, prefix, scheme, suffix = backup.Spec.Gcs.Bucket, backup.Spec.Gcs.Prefix, "gcs", "/"
	case v1alpha1.BackupStorageTypeAzblob:
		bucket, prefix, scheme, suffix = backup.Spec.Azblob.Container, backup.Spec.Azblob.Prefix, "azure", "/"
	case v1alpha1.BackupStorageTypeLocal:
		prefix = backup.Spec.Local.Prefix
		if strings.Trim(prefix, "/") == "" || prefix == "." || prefix == ".." ||
			path.Clean(prefix) != prefix || strings.HasPrefix(prefix, "/") || strings.HasPrefix(prefix, "../") {
			return fmt.Errorf("refusing cleanup: local backup requires a relative non-root prefix")
		}
		return nil
	default:
		return fmt.Errorf("refusing cleanup: unsupported storage type")
	}
	// Match the literal prefix used by the backend, not only path.Join's normalized URL.
	literal := strings.Trim(strings.Trim(bucket, "/")+"/"+strings.Trim(prefix, "/"), "/")
	parts := strings.Split(literal, "/")
	if len(parts) < 2 {
		return fmt.Errorf("refusing cleanup: backup must have a non-empty storage prefix")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("refusing cleanup: storage path is not canonical")
		}
	}
	if target != scheme+"://"+literal+suffix {
		return fmt.Errorf("refusing cleanup: backend target differs from recorded path")
	}
	return nil
}

// Keep deletion anchored to opened directories across symlink changes and renames.
func cleanLocalBackupData(ctx context.Context, backup *v1alpha1.Backup) error {
	mount, err := os.OpenRoot(backup.Spec.Local.VolumeMount.MountPath)
	if err != nil {
		return fmt.Errorf("refusing local cleanup: cannot open mount: %w", err)
	}
	defer mount.Close() //nolint:errcheck
	target, err := mount.OpenRoot(backup.Spec.Local.Prefix)
	if err != nil {
		return fmt.Errorf("refusing local cleanup: cannot open backup within mount: %w", err)
	}
	defer target.Close() //nolint:errcheck
	mountInfo, err := mount.Stat(".")
	if err != nil {
		return err
	}
	targetInfo, err := target.Stat(".")
	if err != nil {
		return err
	}
	if os.SameFile(mountInfo, targetInfo) {
		return fmt.Errorf("refusing local cleanup: backup resolves to the mount root")
	}
	opt := backup.GetCleanOption()
	return util.RetryOnError(ctx, opt.RetryCount, 0, util.RetriableOnAnyError, func() error {
		entries, err := fs.ReadDir(target.FS(), ".")
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := target.RemoveAll(entry.Name()); err != nil {
				return err
			}
		}
		return nil
	})
}
