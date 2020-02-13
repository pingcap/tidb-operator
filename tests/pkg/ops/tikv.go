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

package ops

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	retryLimit            = 15
	maxSSTFilesToTruncate = 20
)

type TruncateOptions struct {
	Namespace string
	Cluster   string
	Store     string
}

type TiKVOps struct {
	ClientOps
}

func (ops *TiKVOps) TruncateSSTFile(opts TruncateOptions) error {
	logHdr := fmt.Sprintf("store: %s cluster: [%s/%s] ", opts.Store, opts.Namespace, opts.Cluster)

	tc, err := ops.PingcapV1alpha1().TidbClusters(opts.Namespace).Get(opts.Cluster, metav1.GetOptions{})
	if err != nil {
		return errors.Trace(err)
	}
	store, ok := tc.Status.TiKV.Stores[opts.Store]
	if !ok {
		return errors.New("no such store")
	}

	exec := func(cmd ...string) (string, string, error) {
		return ops.ExecWithOptions(ExecOptions{
			Command:       cmd,
			Namespace:     opts.Namespace,
			PodName:       store.PodName,
			ContainerName: "tikv",
			CaptureStderr: true,
			CaptureStdout: true,
		})
	}

	retryCount := 0
	for ; retryCount < retryLimit; retryCount++ {
		if retryCount > 0 {
			time.Sleep(10 * time.Second)
		}
		stdout, stderr, err := exec("find", "/var/lib/tikv/db", "-name", "*.sst", "-o", "-name", "*.save")
		if err != nil {
			klog.Warningf(logHdr+"list sst files: stderr=%s err=%s", stderr, err.Error())
			continue
		}

		sstCandidates := make(map[string]bool)

		for _, f := range strings.Split(stdout, "\n") {
			f = strings.TrimSpace(f)
			if len(f) > 0 {
				sstCandidates[f] = true
			}
		}

		ssts := make([]string, 0, maxSSTFilesToTruncate)
		for k := range sstCandidates {
			if len(ssts) >= maxSSTFilesToTruncate {
				break
			}
			if strings.HasSuffix(k, ".sst") && !sstCandidates[k+".save"] {
				ssts = append(ssts, k)
			}
		}
		if len(ssts) == 0 {
			klog.Warning(logHdr + "cannot find a sst file")
			continue
		}

		truncated := 0
		for _, sst := range ssts {
			_, stderr, err = exec("sh", "-c",
				fmt.Sprintf("cp %s %s.save && truncate -s 0 %s", sst, sst, sst))
			if err != nil {
				klog.Warningf(logHdr+"truncate sst file: sst=%s stderr=%s err=%s", sst, stderr, err.Error())
				continue
			}
			truncated++
		}
		if truncated == 0 {
			klog.Warningf(logHdr + "no sst file has been truncated")
			continue
		}

		klog.Infof(logHdr+"%d sst files got truncated", truncated)
		break
	}

	if retryCount == retryLimit {
		return errors.New("failed to truncate sst file after " + strconv.Itoa(retryLimit) + " trials")
	}

	return nil
}

func (ops *TiKVOps) RecoverSSTFile(ns, podName string) error {
	annotateCmd := fmt.Sprintf("kubectl annotate pod %s -n %s runmode=debug --overwrite", podName, ns)
	klog.Info(annotateCmd)
	res, err := exec.Command("/bin/sh", "-c", annotateCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to annotation pod: %s/%s, %v, %s", ns, podName, err, string(res))
	}

	findCmd := fmt.Sprintf("kubectl exec -n %s %s -- find /var/lib/tikv/db -name '*.sst.save'", ns, podName)
	klog.Info(findCmd)
	findData, err := exec.Command("/bin/sh", "-c", findCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to find .save files: %s/%s, %v, %s", ns, podName, err, string(findData))
	}

	for _, saveFile := range strings.Split(string(findData), "\n") {
		saveFile = strings.TrimSpace(saveFile)
		if saveFile == "" {
			continue
		}
		sstFile := strings.TrimSuffix(saveFile, ".save")
		mvCmd := fmt.Sprintf("kubectl exec -n %s %s -- mv %s %s", ns, podName, saveFile, sstFile)
		klog.Info(mvCmd)
		res, err := exec.Command("/bin/sh", "-c", mvCmd).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to recovery .sst files: %s/%s, %s, %s, %v, %s",
				ns, podName, sstFile, saveFile, err, string(res))
		}
	}

	return nil
}
