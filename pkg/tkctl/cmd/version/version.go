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

package version

import (
	"fmt"
	"io"

	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	versionExample = `
		# Print the cli version and tidb operator version
		tkctl version
`
)

type VersionOptions struct {
	ClientOnly bool

	io.Writer
}

func NewCmdVersion(tkcContext *config.TkcContext, out io.Writer) *cobra.Command {
	options := &VersionOptions{
		Writer: out,
	}
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print the client & server version",
		Example: versionExample,
		Run: func(_ *cobra.Command, _ []string) {
			cmdutil.CheckErr(options.runVersion(tkcContext))
		},
	}

	cmd.Flags().BoolVarP(&options.ClientOnly, "client-only", "c", options.ClientOnly,
		"show only client version")

	return cmd
}

func (o *VersionOptions) runVersion(tkcContext *config.TkcContext) error {
	restConfig, err := tkcContext.ToRESTConfig()
	if err != nil {
		return err
	}
	kubeCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	clientVersion := version.Get()

	fmt.Fprintf(o, "Client Version: %s\n", clientVersion)

	if o.ClientOnly {
		return nil
	}

	controllers, err := kubeCli.AppsV1().
		Deployments(core.NamespaceAll).
		List(v1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.ComponentLabelKey, "controller-manager", label.NameLabelKey, "tidb-operator"),
		})
	if err != nil {
		return err
	}
	schedulers, err := kubeCli.AppsV1().
		Deployments(core.NamespaceAll).
		List(v1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s", label.ComponentLabelKey, "scheduler", label.NameLabelKey, "tidb-operator"),
		})
	if err != nil {
		return nil
	}

	// TODO: add version endpoint in tidb-controller-manager
	// There's no version endpoint of tidb-controller-manager and tidb-scheduler, use image instead
	if len(controllers.Items) == 0 {
		fmt.Fprintf(o, "No TiDB Controller Manager found, please install one first\n")
	} else if len(controllers.Items) == 1 {
		fmt.Fprintf(o, "TiDB Controller Manager Version: %s\n", controllers.Items[0].Spec.Template.Spec.Containers[0].Image)
	} else {
		fmt.Fprintf(o, "TiDB Controller Manager Versions:\n")
		for _, item := range controllers.Items {
			fmt.Fprintf(o, "\t%s/%s: %s\n", item.Namespace, item.Name, item.Spec.Template.Spec.Containers[0].Image)
		}
	}
	if len(schedulers.Items) == 0 {
		fmt.Fprintf(o, "No TiDB Scheduler found, please install one first\n")
	} else if len(schedulers.Items) == 1 {
		fmt.Fprintf(o, "TiDB Scheduler Version:\n")
		for _, container := range schedulers.Items[0].Spec.Template.Spec.Containers {
			fmt.Fprintf(o, "\t%s: %s\n", container.Name, container.Image)
		}
	} else {
		// warn for multiple scheduler
		fmt.Fprintf(o, "WARN: more than one TiDB Scheduler deployment found, this is un-supported and may lead to un-expected behavior:\n")
		for _, item := range schedulers.Items {
			fmt.Fprintf(o, "\t%s\n", item.Name)
		}
	}

	return nil
}
