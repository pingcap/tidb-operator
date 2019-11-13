package cmd

import (
	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/to-crdgen/cmd/generate"
	"github.com/spf13/cobra"
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
