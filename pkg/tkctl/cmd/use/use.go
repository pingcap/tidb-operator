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

package use

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	useLongDesc = `
		Specify a tidb cluster to use.
		
		By using a certain cluster, you may omit --tidbcluster option
		in many control commands.
`
	useExample = `
		# specify a tidb cluster to use
		tkctl use demo-cluster

		# specify kubernetes context and namespace
		tkctl use --context=demo-ctx --namespace=demo-ns demo-cluster
`
	useUsage = "expected 'use CLUSTER_NAME' for the use command"
)

type UseOptions struct {
	KubeContext     string
	Namespace       string
	TidbClusterName string

	TcCli *versioned.Clientset

	genericclioptions.IOStreams
}

// NewCmdUse creates the use command.
func NewCmdUse(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	options := &UseOptions{IOStreams: streams}

	cmd := &cobra.Command{
		Use:     "use",
		Short:   "Specify a tidb cluster to use",
		Long:    useLongDesc,
		Example: useExample,
		Run: func(command *cobra.Command, args []string) {
			cmdutil.CheckErr(options.Complete(tkcContext, command, args))
			cmdutil.CheckErr(options.Run(tkcContext, args))
		},
	}

	return cmd
}

func (o *UseOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	o.TidbClusterName = tkcContext.TkcOptions.TidbClusterName
	// 'use' command allows specifying tidb cluster name in args (which overrides options)
	if len(o.TidbClusterName) == 0 && len(args) == 0 {
		return cmdutil.UsageErrorf(cmd, useUsage)
	}
	if len(args) > 0 {
		o.TidbClusterName = args[0]
	}

	kubeConfig := tkcContext.ToRawKubeConfigLoader()

	var err error
	o.Namespace, _, err = kubeConfig.Namespace()
	if err != nil {
		return err
	}

	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return err
	}
	o.TcCli, err = versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// context is private in kubeConfigFlags, so we have to retrieve it from the origin FlagSet
	contextOverride, err := cmd.Flags().GetString("context")
	if err != nil {
		return err
	}
	if len(contextOverride) > 0 {
		o.KubeContext = contextOverride
	} else {
		rawConfig, err := kubeConfig.RawConfig()
		if err != nil {
			return err
		}
		o.KubeContext = rawConfig.CurrentContext
	}
	return nil
}

func (o *UseOptions) Run(tkcContext *config.TkcContext, args []string) error {

	tc, err := o.TcCli.PingcapV1alpha1().
		TidbClusters(o.Namespace).
		Get(o.TidbClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = tkcContext.SwitchTidbCluster(o.KubeContext, tc.Namespace, tc.Name)
	if err == nil {
		fmt.Fprintf(o.Out, "Tidb cluster switched to %s/%s\n", tc.Namespace, tc.Name)
	}
	return err
}
