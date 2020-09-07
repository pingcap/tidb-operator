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

package crypto

import (
	"crypto/x509"
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewCSR(t *testing.T) {
	g := NewGomegaWithT(t)
	cluster := "test-cluster"
	csrBytes, _, err := NewCSR(
		cluster,
		[]string{
			"tidb",
			"tidb.test-cluster.svc",
		}, []string{
			"10.0.0.1",
			"fe80:2333::dead:beef",
		})
	g.Expect(err).Should(BeNil())

	csrObj, err := x509.ParseCertificateRequest(csrBytes)
	g.Expect(err).Should(BeNil())

	g.Expect(csrObj.Subject.CommonName).Should(Equal(cluster))
	g.Expect(csrObj.Subject.Organization).Should(Equal([]string{"PingCAP"}))
	g.Expect(csrObj.Subject.OrganizationalUnit).Should(Equal([]string{"TiDB Operator"}))
	g.Expect(csrObj.DNSNames).Should(Equal(
		[]string{
			"tidb",
			"tidb.test-cluster.svc",
		}))
	g.Expect(csrObj.IPAddresses[0].String()).Should(Equal("10.0.0.1"))
	g.Expect(csrObj.IPAddresses[1].String()).Should(Equal("fe80:2333::dead:beef"))
}
