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

package main

import (
	"flag"
	"fmt"
	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	k8sutil "github.com/pingcap/tidb-operator/pkg/util"
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"os"
)

var (
	cfg crdutils.Config
)

func initFlags(crdkind v1alpha1.CrdKind, flagset *flag.FlagSet) *flag.FlagSet {
	flagset.Var(&cfg.Labels, "labels", "Labels")
	flagset.Var(&cfg.Annotations, "annotations", "Annotations")
	flagset.BoolVar(&cfg.EnableValidation, "with-validation", true, "Add CRD validation field, default: true")
	flagset.StringVar(&cfg.Group, "apigroup", v1alpha1.GroupName, "CRD api group")
	flagset.StringVar(&cfg.SpecDefinitionName, "spec-name", crdkind.SpecName, "CRD spec definition name")
	flagset.StringVar(&cfg.OutputFormat, "output", "yaml", "output format: json|yaml")
	flagset.StringVar(&cfg.Kind, "kind", crdkind.Kind, "CRD Kind")
	flagset.StringVar(&cfg.ResourceScope, "scope", string(extensionsobj.NamespaceScoped), "CRD scope: 'Namespaced' | 'Cluster'.  Default: Namespaced")
	flagset.StringVar(&cfg.Version, "version", v1alpha1.Version, "CRD version, default: 'v1alpha1'")
	flagset.StringVar(&cfg.Plural, "plural", crdkind.Plural, "CRD plural name")
	return flagset
}

func init() {
	var command *flag.FlagSet
	if len(os.Args) == 1 {
		fmt.Println("usage: po-crdgen [tidbcluster | backup | restore | backupschedule [<options>]")
		os.Exit(1)
	}
	switch os.Args[1] {
	case v1alpha1.TiDBClusterKindKey:
		command = initFlags(v1alpha1.DefaultCrdKinds.TiDBCluster, flag.NewFlagSet(v1alpha1.TiDBClusterKindKey, flag.ExitOnError))
	case v1alpha1.BackupKindKey:
		command = initFlags(v1alpha1.DefaultCrdKinds.Backup, flag.NewFlagSet(v1alpha1.BackupKindKey, flag.ExitOnError))
	case v1alpha1.RestoreKindKey:
		command = initFlags(v1alpha1.DefaultCrdKinds.Restore, flag.NewFlagSet(v1alpha1.RestoreKindKey, flag.ExitOnError))
	case v1alpha1.BackupScheduleKindKey:
		command = initFlags(v1alpha1.DefaultCrdKinds.BackupSchedule, flag.NewFlagSet(v1alpha1.BackupScheduleKindKey, flag.ExitOnError))
	default:
		fmt.Printf("%q is not valid command.\n choices: [tidbcluster]", os.Args[1])
		os.Exit(2)
	}
	command.Parse(os.Args[2:])
}

func main() {
	crdKind, err := k8sutil.GetCrdKindFromKindName(cfg.Kind)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	crd := k8sutil.NewCustomResourceDefinition(
		crdKind,
		cfg.Group, cfg.Labels.LabelsMap, cfg.EnableValidation)
	err = crdutils.MarshallCrd(crd, cfg.OutputFormat)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	os.Exit(0)
}
