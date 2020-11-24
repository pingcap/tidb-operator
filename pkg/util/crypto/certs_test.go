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
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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

var certData = []byte(`-----BEGIN CERTIFICATE-----
MIIEMDCCAxigAwIBAgIQUJRs7Bjq1ZxN1ZfvdY+grTANBgkqhkiG9w0BAQUFADCB
gjELMAkGA1UEBhMCVVMxHjAcBgNVBAsTFXd3dy54cmFtcHNlY3VyaXR5LmNvbTEk
MCIGA1UEChMbWFJhbXAgU2VjdXJpdHkgU2VydmljZXMgSW5jMS0wKwYDVQQDEyRY
UmFtcCBHbG9iYWwgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkwHhcNMDQxMTAxMTcx
NDA0WhcNMzUwMTAxMDUzNzE5WjCBgjELMAkGA1UEBhMCVVMxHjAcBgNVBAsTFXd3
dy54cmFtcHNlY3VyaXR5LmNvbTEkMCIGA1UEChMbWFJhbXAgU2VjdXJpdHkgU2Vy
dmljZXMgSW5jMS0wKwYDVQQDEyRYUmFtcCBHbG9iYWwgQ2VydGlmaWNhdGlvbiBB
dXRob3JpdHkwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCYJB69FbS6
38eMpSe2OAtp87ZOqCwuIR1cRN8hXX4jdP5efrRKt6atH67gBhbim1vZZ3RrXYCP
KZ2GG9mcDZhtdhAoWORlsH9KmHmf4MMxfoArtYzAQDsRhtDLooY2YKTVMIJt2W7Q
DxIEM5dfT2Fa8OT5kavnHTu86M/0ay00fOJIYRyO82FEzG+gSqmUsE3a56k0enI4
qEHMPJQRfevIpoy3hsvKMzvZPTeL+3o+hiznc9cKV6xkmxnr9A8ECIqsAxcZZPRa
JSKNNCyy9mgdEm3Tih4U2sSPpuIjhdV6Db1q4Ons7Be7QhtnqiXtRYMh/MHJfNVi
PvryxS3T/dRlAgMBAAGjgZ8wgZwwEwYJKwYBBAGCNxQCBAYeBABDAEEwCwYDVR0P
BAQDAgGGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFMZPoj0GY4QJnM5i5ASs
jVy16bYbMDYGA1UdHwQvMC0wK6ApoCeGJWh0dHA6Ly9jcmwueHJhbXBzZWN1cml0
eS5jb20vWEdDQS5jcmwwEAYJKwYBBAGCNxUBBAMCAQEwDQYJKoZIhvcNAQEFBQAD
ggEBAJEVOQMBG2f7Shz5CmBbodpNl2L5JFMn14JkTpAuw0kbK5rc/Kh4ZzXxHfAR
vbdI4xD2Dd8/0sm2qlWkSLoC295ZLhVbO50WfUfXN+pfTXYSNrsf16GBBEYgoyxt
qZ4Bfj8pzgCT3/3JknOJiWSe5yvkHJEs0rnOfc5vMZnT5r7SHpDwCRR5XCOrTdLa
IR9NmXmd4c8nnxCbHIgNsIpkQTG4DmyQJKSbXHGPurt+HBvbaoAPIbzp26a3QPSy
i6mx5O+aGtA9aZnuqCij4Tyz8LIRnM98QObd50N9otg6tamN8jSZxNQQ4Qb9CYQQ
O+7ETPTsJ3xCwnR8gooJybQDJbw=
-----END CERTIFICATE-----`)

func TestReadCACerts(t *testing.T) {
	g := NewGomegaWithT(t)

	// write the ca file for test
	f, err := ioutil.TempFile("", "")
	defer os.Remove(f.Name())
	g.Expect(err).Should(BeNil())
	_, err = f.Write(certData)
	g.Expect(err).Should(BeNil())

	_, err = readCACerts(f.Name())
	g.Expect(err).Should(BeNil())
}

func TestTlsConfigFromSecret(t *testing.T) {
	certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)
	keyPem := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)

	g := NewGomegaWithT(t)
	secret := &corev1.Secret{
		Data: make(map[string][]byte),
	}

	// test all error case and normal case finally.
	_, err := LoadTlsConfigFromSecret(secret)
	g.Expect(err).Should(MatchError("failed to append ca certs"))

	secret.Data[corev1.ServiceAccountRootCAKey] = certData
	_, err = LoadTlsConfigFromSecret(secret)
	g.Expect(err.Error()).Should(MatchRegexp("cert or key does not exist.*"))

	secret.Data[corev1.TLSCertKey] = append(certPem, []byte("messy up data")...)
	secret.Data[corev1.TLSPrivateKeyKey] = keyPem
	_, err = LoadTlsConfigFromSecret(secret)
	g.Expect(err.Error()).Should(MatchRegexp("unable to load certificates from secret.*"))

	secret.Data[corev1.TLSCertKey] = certPem
	secret.Data[corev1.TLSPrivateKeyKey] = keyPem
	_, err = LoadTlsConfigFromSecret(secret)
	g.Expect(err).Should(BeNil())
}
