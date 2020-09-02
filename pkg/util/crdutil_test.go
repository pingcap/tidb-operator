// Copyright 2018 PingCAP, Inc.
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

package util

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func TestNewCustomResourceDefinition(t *testing.T) {
	g := NewGomegaWithT(t)

	kind, err := GetCrdKindFromKindName("tidbcluster")
	g.Expect(err).Should(Succeed())
	crd := NewCustomResourceDefinition(
		kind,
		"test-group",
		map[string]string{"test": "ut"},
		false,
	)
	g.Expect(crd).ShouldNot(BeNil())
	g.Expect(crd.Spec.Group).Should(Equal("test-group"))
	g.Expect(crd.ObjectMeta.Labels["test"]).Should(Equal("ut"))
	g.Expect(crd.Spec.AdditionalPrinterColumns).Should(Equal(tidbClusteradditionalPrinterColumns))
}

func TestGetCrdKindFromKindName(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(GetCrdKindFromKindName("tidbCluster")).
		Should(Equal(v1alpha1.DefaultCrdKinds.TiDBCluster))
	g.Expect(GetCrdKindFromKindName("DMcluster")).
		Should(Equal(v1alpha1.DefaultCrdKinds.DMCluster))
	g.Expect(GetCrdKindFromKindName("backup")).
		Should(Equal(v1alpha1.DefaultCrdKinds.Backup))
	g.Expect(GetCrdKindFromKindName("restore")).
		Should(Equal(v1alpha1.DefaultCrdKinds.Restore))
	g.Expect(GetCrdKindFromKindName("backupSchedule")).
		Should(Equal(v1alpha1.DefaultCrdKinds.BackupSchedule))
	g.Expect(GetCrdKindFromKindName("TiDBMonitor")).
		Should(Equal(v1alpha1.DefaultCrdKinds.TiDBMonitor))
	g.Expect(GetCrdKindFromKindName("tidbinitializer")).
		Should(Equal(v1alpha1.DefaultCrdKinds.TiDBInitializer))
	g.Expect(GetCrdKindFromKindName("TidbClusterAutoScaler")).
		Should(Equal(v1alpha1.DefaultCrdKinds.TidbClusterAutoScaler))
	g.Expect(GetCrdKindFromKindName("tikvgroup")).
		Should(Equal(v1alpha1.DefaultCrdKinds.TiKVGroup))
	g.Expect(GetCrdKindFromKindName("tidbgroup")).
		Should(Equal(v1alpha1.DefaultCrdKinds.TiDBGroup))
	_, err := GetCrdKindFromKindName("pingcap")
	g.Expect(err).
		Should(MatchError("unknown CrdKind Name"))
}
