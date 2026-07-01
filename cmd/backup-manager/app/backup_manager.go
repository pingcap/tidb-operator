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
	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
)

func Run() error {
	logs.InitLogs()
	defer logs.FlushLogs()

	// fix klog parse error
	flag.CommandLine.Parse([]string{})

	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	if err := pflag.Set("logtostderr", "true"); err != nil {
		panic("failed to set logtostderr flag: " + err.Error())
	}
	// Opt into the fixed klog behavior so the --stderrthreshold flag is honored
	// even when --logtostderr is enabled. See https://github.com/kubernetes/klog/issues/432
	if err := pflag.Set("legacy_stderr_threshold_behavior", "false"); err != nil {
		panic("failed to set legacy_stderr_threshold_behavior flag: " + err.Error())
	}
	if err := pflag.Set("stderrthreshold", "INFO"); err != nil {
		panic("failed to set stderrthreshold flag: " + err.Error())
	}
	// We do not want these flags to show up in --help
	// These MarkHidden calls must be after the lines above
	pflag.CommandLine.MarkHidden("version")
	pflag.CommandLine.MarkHidden("google-json-key")
	pflag.CommandLine.MarkHidden("log-flush-frequency")
	pflag.CommandLine.MarkHidden("alsologtostderr")
	pflag.CommandLine.MarkHidden("log-backtrace-at")
	pflag.CommandLine.MarkHidden("log-dir")
	pflag.CommandLine.MarkHidden("logtostderr")
	pflag.CommandLine.MarkHidden("stderrthreshold")
	pflag.CommandLine.MarkHidden("vmodule")
	command := cmd.NewBackupMgrCommand()
	return command.Execute()
}
