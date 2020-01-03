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

	"github.com/stretchr/testify/assert"
	glog "k8s.io/klog"
)

var (
	t = MyTestT{}
)

type MyTestT struct{}

func (mt *MyTestT) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args)
}

func newTkctlCmd(args []string) *exec.Cmd {
	return exec.Command("tkctl", args...)
}

func checkListOrDie(info *TidbClusterConfig) {
	output, err := newTkctlCmd([]string{"list", "--namespace", info.Namespace}).Output()
	if assert.Nil(&t, err) {
		glog.Fatalf("command 'list' not run as expect %v", err)
	}
	pdnum := info.Resources["pd.replicas"]
	kvnum := info.Resources["tikv.replicas"]
	dbnum := info.Resources["tidb.replicas"]
	restr := fmt.Sprintf(".*%s/%s.*%s/%s.*%s/%s.*",pdnum,pdnum,kvnum,kvnum,dbnum,dbnum)
	if assert.Regexp(&t, regexp.MustCompile(restr), string(output)) {
		glog.Fatalf("command 'list' not run as expect")
	}
}

func checkUseOrDie(info *TidbClusterConfig) {
	output, err := newTkctlCmd([]string{"use", info.ClusterName}).Output()
	if assert.Nil(&t, err) {
		glog.Fatalf("command 'use' not run as expect %v", err)
	}
	if assert.Regexp(&t, regexp.MustCompile(fmt.Sprintf("switched to %s/%s", info.Namespace, info.ClusterName)), string(output)) {
		glog.Fatalf("command 'use' not run as expect")
	}
}

func (oa *operatorActions) CheckTkctlOrDie(info *TidbClusterConfig) {
	checkListOrDie(info)
	checkUseOrDie(info)
}
