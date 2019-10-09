package util

import (
	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func NewCustomResourceDefinition(crdKind v1alpha1.CrdKind, group string, labels map[string]string, validation bool) *extensionsobj.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		SpecDefinitionName:    crdKind.SpecName,
		EnableValidation:      validation,
		Labels:                crdutils.Labels{LabelsMap: labels},
		ResourceScope:         string(extensionsobj.NamespaceScoped),
		Group:                 group,
		Kind:                  crdKind.Kind,
		Version:               v1alpha1.Version,
		Plural:                crdKind.Plural,
		ShortNames:            crdKind.ShortNames,
		GetOpenAPIDefinitions: v1alpha1.GetOpenAPIDefinitions,
	})
}
