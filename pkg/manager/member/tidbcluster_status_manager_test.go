// Copyright 2020 PingCAP, Inc.
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

package member

import (
	"fmt"
	"regexp"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

func TestTidbPattern(t *testing.T) {
	g := NewGomegaWithT(t)

	name := "basic"
	ns := "tidb-cluster"

	// test no domain
	reg := fmt.Sprintf(tidbAddrPattern, name, name, ns, controller.FormatClusterDomainForRegex(""))
	pattern, err := regexp.Compile(reg)
	g.Expect(err).Should(BeNil())

	m := pattern.Match([]byte("basic-tidb-0.basic-tidb-peer.tidb-cluster.svc"))
	g.Expect(m).Should(BeTrue())

	m = pattern.Match([]byte("basic-tidb-0.basic-tidb-peer.tidb-cluster.svc.other.domain"))
	g.Expect(m).Should(BeFalse())

	m = pattern.Match([]byte("othername-tidb-0.basic-tidb-peer.tidb-cluster.svc.other.domain"))
	g.Expect(m).Should(BeFalse())

	m = pattern.Match([]byte("othername-tidb-0.basic-tidb-peer.otherns.svc.other.domain"))
	g.Expect(m).Should(BeFalse())

	// test with domain
	reg = fmt.Sprintf(tidbAddrPattern, name, name, ns, controller.FormatClusterDomainForRegex("d1.d2"))
	pattern, err = regexp.Compile(reg)
	g.Expect(err).Should(BeNil())

	m = pattern.Match([]byte("basic-tidb-0.basic-tidb-peer.tidb-cluster.svc.d1.d2"))
	g.Expect(m).Should(BeTrue())

	// domain is optional
	m = pattern.Match([]byte("basic-tidb-0.basic-tidb-peer.tidb-cluster.svc"))
	g.Expect(m).Should(BeTrue())

	m = pattern.Match([]byte("basic-tidb-0.basic-tidb-peer.tidb-cluster.svc.other.domain"))
	g.Expect(m).Should(BeFalse())

	m = pattern.Match([]byte("othername-tidb-0.basic-tidb-peer.tidb-cluster.svc.other.d1.d2"))
	g.Expect(m).Should(BeFalse())

	m = pattern.Match([]byte("othername-tidb-0.basic-tidb-peer.otherns.svc.other.d1.d2"))
	g.Expect(m).Should(BeFalse())
}
