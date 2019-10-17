// Copyright 2019. PingCAP, Inc.
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

package main

import (
	"flag"
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/to-crdgen/cmd"
	"github.com/spf13/pflag"
	"os"
)

func main() {
	flags := pflag.NewFlagSet("to-crdgen", pflag.ExitOnError)
	flag.CommandLine.Parse([]string{})
	pflag.CommandLine = flags

	command := cmd.NewToCrdGenRootCmd()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
