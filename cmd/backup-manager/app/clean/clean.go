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

	"k8s.io/klog"

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
func (bo *Options) cleanBRRemoteBackupData(backup *v1alpha1.Backup) error {
	s, err := util.NewRemoteStorage(backup)
	if err != nil {
		return err
	}
	defer s.Close()
	ctx := context.Background()
	iter := s.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		klog.Infof("Prepare to delete %s for cluster %s", obj.Key, bo)
		err = s.Delete(context.Background(), obj.Key)
		if err != nil {
			return err
		}
		klog.Infof("Delete %s for cluster %s successfully", obj.Key, bo)
	}
	return nil
}

func (bo *Options) cleanRemoteBackupData(bucket string, opts []string) error {
	destBucket := util.NormalizeBucketURI(bucket)
	args := util.ConstructArgs(constants.RcloneConfigArg, opts, "deletefile", destBucket, "")
	output, err := exec.Command("rclone", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone deletefile command failed, output: %s, err: %v", bo, string(output), err)
	}

	klog.Infof("cluster %s backup %s was deleted successfully", bo, bucket)
	return nil
}
