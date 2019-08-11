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

package app

import (
	"flag"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/cmd"
	"k8s.io/apiserver/pkg/util/logs"
)

func Run() error {
	logs.InitLogs()
	defer logs.FlushLogs()
	flag.CommandLine.Parse([]string{})

	command := cmd.NewBackupMgrCommand()
	return command.Execute()
}
