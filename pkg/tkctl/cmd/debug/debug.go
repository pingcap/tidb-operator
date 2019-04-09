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

package debug

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
)

const (
	example = `
	# debug a container in the running pod, the first container will be picked by default
	kubectl debug POD_NAME

	# specify namespace or container
	kubectl debug --namespace foo POD_NAME -c CONTAINER_NAME

	# override the default troubleshooting image
	kubectl debug POD_NAME --image aylei/debug-jvm

	# override entrypoint of debug container
	kubectl debug POD_NAME --image aylei/debug-jvm /bin/bash

`
	longDesc = `
Run a container in a running pod, this container will join the namespaces of an existing container of the pod.
`
	defaultImage = "nicolaka/netshoot:latest"
)

// TODO: implementation
// NewCmdDebug creates the debug subcommand which helps container debugging
func NewCmdDebug(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "debug",
		Short: "Not implemented",
		Run: func(_ *cobra.Command, args []string) {
			fmt.Fprint(os.Stdout, "not implemented")
		},
	}
}

// DebugOptions specify how to run debug container in a running pod
type DebugOptions struct {

	// Pod select options
	Namespace string
	PodName   string

	// Debug options
	Image         string
	ContainerName string
	Command       []string
}

// Complete populate default values for DebugOptions
func (p *DebugOptions) Complete(configFlags *genericclioptions.ConfigFlags, cmd *cobra.Command, argsIn []string, argsLenAtDash int) error {
	// select one pod to debug (required)

	// we have to define kv,db,pd selector here (tidb-context aware)

	// override default image, container, command
	return nil
}

func (p *DebugOptions) Validate() error {
	// validate required fields are populated
	return nil
}

func (p *DebugOptions) Run() error {
	// Get Pod and verify state

	// Create launch pod, wait remote shell attached and running

	// Hold and cleanup
	return nil
}
