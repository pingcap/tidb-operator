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

package tests

import (
	"fmt"
	"os/exec"
	"regexp"

	glog "k8s.io/klog"
)


func newTkctlCmd(args []string) *exec.Cmd {
	return exec.Command("tkctl", args...)
}

func CheckListOrDie(info *TidbClusterConfig) {
	output, err := newTkctlCmd([]string{"list", "--namespace", info.Namespace}).Output()
	if err != nil {
		glog.Fatalf("command 'list' not run as expect %v", err)
	}
	pdnum := info.Resources["pd.replicas"]
	kvnum := info.Resources["tikv.replicas"]
	dbnum := info.Resources["tidb.replicas"]
	restr := fmt.Sprintf(".*%s/%s.*%s/%s.*%s/%s.*",pdnum,pdnum,kvnum,kvnum,dbnum,dbnum)
	re := regexp.MustCompile(restr)
	if !re.Match(output) {
		glog.Fatalf("command 'list' not run as expect, expect %s, but got %s",restr,output)
	}
}

func CheckUseOrDie(info *TidbClusterConfig) {
	output, err := newTkctlCmd([]string{"use", info.ClusterName}).Output()
	if err != nil {
		glog.Fatalf("command 'use' not run as expect %v", err)
	}
	restr := fmt.Sprintf("switched to %s/%s", info.Namespace, info.ClusterName)
	re := regexp.MustCompile(restr)
	if !re.Match(output) {
		glog.Fatalf("command 'use' not run as expect, expect %s, but got %s",restr,output)
	}
}
