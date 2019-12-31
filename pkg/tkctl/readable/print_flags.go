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

package readable

import (
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kubeprinters "k8s.io/kubernetes/pkg/printers"
)

type PrintFlags struct {
	JSONYamlPrintFlags *genericclioptions.JSONYamlPrintFlags
	OutputFormat       string
}

func NewPrintFlags() *PrintFlags {
	return &PrintFlags{
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	}
}

func (p *PrintFlags) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.OutputFormat, "output", "o", p.OutputFormat,
		"Output format. json|yaml|wide")
	p.JSONYamlPrintFlags.AddFlags(cmd)
}

func (p *PrintFlags) ToPrinter(withKind, withNamespace bool) (kubeprinters.ResourcePrinter, error) {
	output := strings.ToLower(p.OutputFormat)
	if output == "json" || output == "yaml" {
		printer, err := p.JSONYamlPrintFlags.ToPrinter(output)
		if err != nil {
			return nil, err
		}
		return printer, nil
	}
	printer := kubeprinters.NewTablePrinter(kubeprinters.PrintOptions{
		WithNamespace: withNamespace,
		Wide:          p.OutputFormat == "wide",
		WithKind:      withKind,
	})
	tableGenerator := kubeprinters.NewTableGenerator().With(AddHandlers)
	options := kubeprinters.GenerateOptions{
		Wide: p.OutputFormat == "wide",
	}
	return NewLocalPrinter(printer, tableGenerator, options), nil
}
