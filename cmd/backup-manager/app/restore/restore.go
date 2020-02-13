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
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"

	backupUtil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type Options struct {
	backupUtil.GenericOptions
}

func (ro *Options) restoreData(restore *v1alpha1.Restore) error {
	clusterNamespace := restore.Spec.BR.ClusterNamespace
	if restore.Spec.BR.ClusterNamespace == "" {
		clusterNamespace = restore.Namespace
	}
	args, err := constructBROptions(restore)
	if err != nil {
		return err
	}
	args = append(args, fmt.Sprintf("--pd=%s-pd.%s:2379", restore.Spec.BR.Cluster, clusterNamespace))
	if ro.TLSCluster {
		args = append(args, fmt.Sprintf("--ca=%s", path.Join(util.ClusterClientTLSPath, corev1.ServiceAccountRootCAKey)))
		args = append(args, fmt.Sprintf("--cert=%s", path.Join(util.ClusterClientTLSPath, corev1.TLSCertKey)))
		args = append(args, fmt.Sprintf("--key=%s", path.Join(util.ClusterClientTLSPath, corev1.TLSPrivateKeyKey)))
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
	klog.Infof("Running br command with args: %v", fullArgs)
	bin := "br" + backupUtil.Suffix(ro.TiKVVersion)
	cmd := exec.Command(bin, fullArgs...)

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stdout pipe failed, err: %v", ro, err)
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stderr pipe failed, err: %v", ro, err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cluster %s, execute br command failed, args: %s, err: %v", ro, fullArgs, err)
	}
	var errMsg string
	reader := bufio.NewReader(stdOut)
	for {
		line, err := reader.ReadString('\n')
		if strings.Contains(line, "[ERROR]") {
			errMsg += line
		}
		klog.Infof(strings.Replace(line, "\n", "", -1))
		if err != nil || io.EOF == err {
			break
		}
	}
	tmpErr, _ := ioutil.ReadAll(stdErr)
	if len(tmpErr) > 0 {
		klog.Infof(string(tmpErr))
		errMsg += string(tmpErr)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("cluster %s, wait pipe message failed, errMsg %s, err: %v", ro, errMsg, err)
	}
	klog.Infof("Restore data for cluster %s successfully", ro)
	return nil
}

func constructBROptions(restore *v1alpha1.Restore) ([]string, error) {
	args, err := backupUtil.ConstructBRGlobalOptionsForRestore(restore)
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
