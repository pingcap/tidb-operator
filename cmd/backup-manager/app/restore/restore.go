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

package restore

import (
	"fmt"
	"os/exec"
	"path"

	glog "k8s.io/klog"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

type Options struct {
	Namespace   string
	RestoreName string
}

func (ro *Options) String() string {
	return fmt.Sprintf("%s/%s", ro.Namespace, ro.RestoreName)
}

func (ro *Options) restoreData(restore *v1alpha1.Restore) error {
	var restoreNamespace string
	args, err := constructBROptions(restore)
	if err != nil {
		return err
	}
	if restore.Spec.RestoreNamespace == "" {
		restoreNamespace = ro.Namespace
	} else {
		restoreNamespace = restore.Spec.RestoreNamespace
	}
	if restore.Spec.EnableTLSClient {
		args = append(args, fmt.Sprintf("--pd=https://%s-pd.%s", restore.Spec.Cluster, restoreNamespace))
		args = append(args, fmt.Sprintf("--ca=%s", constants.ServiceAccountCAPath))
		args = append(args, fmt.Sprintf("--cert=%s", path.Join(constants.BRCertPath, "cert")))
		args = append(args, fmt.Sprintf("--key=%s", path.Join(constants.BRCertPath, "key")))
	} else {
		args = append(args, fmt.Sprintf("--pd=http://%s-pd.%s", restore.Spec.Cluster, restoreNamespace))
	}

	var restoreType string
	if restore.Spec.Type == "" {
		restoreType = string(v1alpha1.BackupTypeFull)
	} else {
		restoreType = string(restore.Spec.Type)
	}
	fullArgs := []string{
		"restore",
		restoreType,
	}
	fullArgs = append(fullArgs, args...)
	glog.Infof("Running br command with args: %v", fullArgs)
	output, err := exec.Command("br", fullArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute br command %v failed, output: %s, err: %v", ro, fullArgs, string(output), err)
	}
	glog.Infof("Restore data for cluster %s successfully, output: %s", ro, string(output))
	return nil
}

func constructBROptions(restore *v1alpha1.Restore) ([]string, error) {
	args, err := util.ConstructBRGlobalOptionsForRestore(restore)
	if err != nil {
		return nil, err
	}
	config := restore.Spec.BR
	if config.Concurrency != nil {
		args = append(args, fmt.Sprintf("--concurrency=%d", *config.Concurrency))
	}
	if config.Checksum != nil {
		args = append(args, fmt.Sprintf("--checksum=%t", *config.Checksum))
	}
	if config.RateLimit != nil {
		args = append(args, fmt.Sprintf("--ratelimit=%d", *config.RateLimit))
	}
	if config.OnLine != nil {
		args = append(args, fmt.Sprintf("--online=%t", *config.OnLine))
	}
	return args, nil
}
