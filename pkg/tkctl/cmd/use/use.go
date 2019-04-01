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

package use

import (
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	useLongDesc = `
		Specify a tidb cluster to use.
		
		By using a certain cluster, you may omit --tidbcluster option or <clusterName>
		option in many control commands.
`
	useExample = `
		# specify a tidb cluster to use
		tkc use demo-cluster

		# specify kubernetes context and namespace
		tkc use --context=demo-ctx --namespace=demo-ns demo-cluster
`
	useUsage = "expected 'use CLUSTER_NAME' for the use command"
)

type UseOptions struct {
	KubeContext     string
	Namespace       string
	TidbClusterName string
}

// NewCmdUse creates the use command.
func NewCmdUse(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	options := &UseOptions{}

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

	cmd.Flags().StringVarP(&options.TidbClusterName, "tidbcluster", "t", options.TidbClusterName, "Tidb cluster name")

	return cmd
}

func (o *UseOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, args []string) error {
	if len(o.TidbClusterName) == 0 && len(args) == 0 {
		return cmdutil.UsageErrorf(cmd, useUsage)
	}
	if len(o.TidbClusterName) == 0 {
		o.TidbClusterName = args[0]
	}

	kubeConfig := tkcContext.ToRawKubeConfigLoader()

	var err error
	o.Namespace, _, err = kubeConfig.Namespace()
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

	// For the 'use' command, typically the user wants to switch to another tidb cluster
	// in a different kube-context or namespace, so it's better to use the raw kubectl context
	restConfig, err := tkcContext.ToKubectlRestConfig()
	if err != nil {
		return err
	}
	tcCli, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	tc, err := tcCli.PingcapV1alpha1().
		TidbClusters(o.Namespace).
		Get(o.TidbClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return tkcContext.SwitchTidbCluster(o.KubeContext, tc.Namespace, tc.Name)
}
