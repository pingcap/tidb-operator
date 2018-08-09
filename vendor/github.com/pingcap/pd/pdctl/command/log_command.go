// Copyright 2018 PingCAP, Inc.
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

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var (
	logPrefix = "pd/api/v1/log"
)

// NewLogCommand New a log subcommand of the rootCmd
func NewLogCommand() *cobra.Command {
	conf := &cobra.Command{
		Use:   "log [fatal|error|warn|info|debug]",
		Short: "set log level",
		Run:   logCommandFunc,
	}
	return conf
}

func logCommandFunc(cmd *cobra.Command, args []string) {
	var err error
	if len(args) != 1 {
		fmt.Println(cmd.UsageString())
		return
	}

	data, err := json.Marshal(args[0])
	if err != nil {
		fmt.Printf("Failed to set log level: %s\n", err)
		return
	}
	req, err := getRequest(cmd, logPrefix, http.MethodPost, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("Failed to set log level: %s\n", err)
		return
	}
	_, err = dail(req)
	if err != nil {
		fmt.Printf("Failed to set log level: %s\n", err)
		return
	}
	fmt.Println("Success!")
}
