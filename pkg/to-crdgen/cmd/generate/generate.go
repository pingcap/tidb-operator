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

package generate

import (
	"errors"
	crdutils "github.com/yisaer/crd-validation/pkg"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	k8sutil "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	usage = "usage: to-crdgen generate [tidbcluster | backup | restore | backupschedule [<options>]"
)

func AddGenerateCommand(config *crdutils.Config) *cobra.Command {
	generatedCommand := &cobra.Command{
		Use:   "generate",
		Short: "Generate CRD",
		Long:  "Generate CRD by crd-util according to types",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(generate(config, args))
		},
	}
	return generatedCommand
}

func initConfig(kind v1alpha1.CrdKind, config *crdutils.Config) {
	config.Kind = kind.Kind
	config.Plural = kind.Plural
	config.ShortNames = kind.ShortNames
	config.SpecDefinitionName = kind.SpecName
}

func generate(config *crdutils.Config, args []string) error {
	if len(args) < 1 || len(args) > 1 {
		return errors.New(usage)
	}
	crdKind, err := k8sutil.GetCrdKindFromKindName(args[0])
	if err != nil {
		return errors.New(usage)
	}
	initConfig(crdKind, config)
	crd := k8sutil.NewCustomResourceDefinition(
		crdKind,
		config.Group, config.Labels.LabelsMap, config.EnableValidation)
	return crdutils.MarshallCrd(crd, config.OutputFormat)
}
