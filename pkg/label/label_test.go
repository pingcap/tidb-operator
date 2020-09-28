// Copyright 2017 PingCAP, Inc.
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

package label

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
)

func TestLabelNew(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	g.Expect(l[NameLabelKey]).To(Equal("tidb-cluster"))
	g.Expect(l[ManagedByLabelKey]).To(Equal("tidb-operator"))
}

func TestLabelNewDM(t *testing.T) {
	g := NewGomegaWithT(t)

	l := NewDM()
	g.Expect(l[NameLabelKey]).To(Equal("dm-cluster"))
	g.Expect(l[ManagedByLabelKey]).To(Equal("tidb-operator"))
}

func TestLabelInstance(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.Instance("demo")
	g.Expect(l[InstanceLabelKey]).To(Equal("demo"))
}

func TestLabelNamespace(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.Namespace("ns-1")
	g.Expect(l[NamespaceLabelKey]).To(Equal("ns-1"))
}

func TestLabelComponent(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.Component("tikv")
	g.Expect(l.ComponentType()).To(Equal("tikv"))
}

func TestLabelPD(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.PD()
	g.Expect(l.IsPD()).To(BeTrue())
}

func TestLabelTiDB(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.TiDB()
	g.Expect(l.IsTiDB()).To(BeTrue())
}

func TestLabelTiKV(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.TiKV()
	g.Expect(l.IsTiKV()).To(BeTrue())
}

func TestLabelDMMaster(t *testing.T) {
	g := NewGomegaWithT(t)

	l := NewDM()
	l.DMMaster()
	g.Expect(l.IsDMMaster()).To(BeTrue())
}

func TestLabelDMWorker(t *testing.T) {
	g := NewGomegaWithT(t)

	l := NewDM()
	l.DMWorker()
	g.Expect(l.IsDMWorker()).To(BeTrue())
}

func TestLabelSelector(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.PD()
	l.Instance("demo")
	l.Namespace("ns-1")
	s, err := l.Selector()
	st := labels.Set(map[string]string{
		NameLabelKey:      "tidb-cluster",
		ManagedByLabelKey: "tidb-operator",
		ComponentLabelKey: "pd",
		InstanceLabelKey:  "demo",
		NamespaceLabelKey: "ns-1",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s.Matches(st)).To(BeTrue())
}

func TestLabelLabelSelector(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.PD()
	l.Instance("demo")
	l.Namespace("ns-1")
	ls := l.LabelSelector()
	m := map[string]string{
		NameLabelKey:      "tidb-cluster",
		ManagedByLabelKey: "tidb-operator",
		ComponentLabelKey: "pd",
		InstanceLabelKey:  "demo",
		NamespaceLabelKey: "ns-1",
	}
	g.Expect(ls.MatchLabels).To(Equal(m))
}

func TestLabelLabels(t *testing.T) {
	g := NewGomegaWithT(t)

	l := New()
	l.PD()
	l.Instance("demo")
	l.Namespace("ns-1")
	ls := l.Labels()
	m := map[string]string{
		NameLabelKey:      "tidb-cluster",
		ManagedByLabelKey: "tidb-operator",
		ComponentLabelKey: "pd",
		InstanceLabelKey:  "demo",
		NamespaceLabelKey: "ns-1",
	}
	g.Expect(ls).To(Equal(m))
}

func TestDMLabelLabels(t *testing.T) {
	g := NewGomegaWithT(t)

	l := NewDM()
	l.DMMaster()
	l.Instance("demo")
	l.Namespace("ns-1")
	ls := l.Labels()
	m := map[string]string{
		NameLabelKey:      "dm-cluster",
		ManagedByLabelKey: "tidb-operator",
		ComponentLabelKey: "dm-master",
		InstanceLabelKey:  "demo",
		NamespaceLabelKey: "ns-1",
	}
	g.Expect(ls).To(Equal(m))
}
