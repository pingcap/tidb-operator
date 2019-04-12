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
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	glog.Infof("truncate sst option: %+v", opts)

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

	stdout, stderr, err := exec("find", "/var/lib/tikv/db", "-name", "*.sst", "-o", "-name", "*.save")
	if err != nil {
		glog.Errorf("list sst files: stderr=%s err=%s", stderr, err.Error())
		return errors.Annotate(err, "list sst files")
	}

	sstCandidates := make(map[string]bool)

	for _, f := range strings.Split(stdout, "\n") {
		f = strings.TrimSpace(f)
		if len(f) > 0 {
			sstCandidates[f] = true
		}
	}

	sst := ""
	for k := range sstCandidates {
		if strings.HasSuffix(k, ".sst") && !sstCandidates[k+".save"] {
			sst = k
		}
	}
	if len(sst) == 0 {
		return errors.New("cannot find a sst file")
	}

	_, stderr, err = exec("cp", sst, sst+".save")
	if err != nil {
		glog.Errorf("backup sst file: stderr=%s err=%s", stderr, err.Error())
		return errors.Annotate(err, "backup sst file")
	}

	_, stderr, err = exec("truncate", "-s", "0", sst)
	if err != nil {
		glog.Errorf("truncate sst file: stderr=%s err=%s", stderr, err.Error())
		return errors.Annotate(err, "truncate sst file")
	}

	return nil
}
