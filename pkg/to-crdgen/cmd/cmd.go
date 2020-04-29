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

package cmd

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/to-crdgen/cmd/generate"
	"github.com/spf13/cobra"
	crdutils "github.com/yisaer/crd-validation/pkg"
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	tkcLongDescription = `
		"to-crdgen (TiDB-Operator crd generator) is a tool to help generate CRD automatically.
`
)

var (
	cfg crdutils.Config
)

func initFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&cfg.EnableValidation, "with-validation", true, "Add CRD validation field, default: true")
	cmd.Flags().StringVar(&cfg.Group, "apigroup", v1alpha1.GroupName, "CRD api group")
	cmd.Flags().StringVar(&cfg.OutputFormat, "output", "yaml", "output format: json|yaml")
	cmd.Flags().StringVar(&cfg.ResourceScope, "scope", string(extensionsobj.NamespaceScoped), "CRD scope: 'Namespaced' | 'Cluster'.  Default: Namespaced")
	cmd.Flags().StringVar(&cfg.Version, "version", v1alpha1.Version, "CRD version, default: 'v1alpha1'")
}

func NewToCrdGenRootCmd() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "to-crdgen",
		Short: "to-crdgen is a small tool to generate crd",
		Long:  tkcLongDescription,
		Run:   runHelp,
	}
	initFlags(rootCmd)
	rootCmd.AddCommand(generate.AddGenerateCommand(&cfg))
	return rootCmd
}

func runHelp(cmd *cobra.Command, _ []string) {
	cmd.Help()
}
