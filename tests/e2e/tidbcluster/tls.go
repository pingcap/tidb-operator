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

package tidbcluster

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"os"
	"os/exec"
	"text/template"
)

var tidbIssuerTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: {{ .ClusterName }}-selfsigned-ca-issuer
  namespace: {{ .Namespace }}
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-ca
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-ca-secret
  commonName: "TiDB CA"
  isCA: true
  issuerRef:
    name: {{ .ClusterRef }}-selfsigned-ca-issuer
    kind: Issuer
---
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: {{ .ClusterName }}-tidb-issuer
  namespace: {{ .Namespace }}
spec:
  ca:
    secretName: {{ .ClusterName }}-ca-secret
`

var tidbCertificatesTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-tidb-server-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-tidb-server-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
    - PingCAP
  commonName: "TiDB Server"
  usages:
    - server auth
  dnsNames:
    - "{{ .ClusterName }}-tidb"
    - "{{ .ClusterName }}-tidb.{{ .Namespace }}"
    - "*.{{ .ClusterName }}-tidb"
    - "{{ .ClusterName }}-tidb.{{ .Namespace }}.svc"
    - "*.{{ .ClusterName }}-tidb.{{ .Namespace }}"
    - "*.{{ .ClusterName }}-tidb.{{ .Namespace }}.svc"
  ipAddresses:
    - 127.0.0.1
    - ::1
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-tidb-client-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-tidb-client-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
    - PingCAP
  commonName: "TiDB Client"
  usages:
    - client auth
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
`

var tidbComponentsCertificatesTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-pd-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-pd-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-pd"
  - "{{ .ClusterName }}-pd.{{ .Namespace }}"
  - "{{ .ClusterName }}-pd.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-pd-peer"
  - "{{ .ClusterName }}-pd-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-pd-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-pd-peer"
  - "*.{{ .ClusterName }}-pd-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-pd-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-tikv-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-tikv-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-tikv"
  - "{{ .ClusterName }}-tikv.{{ .Namespace }}"
  - "{{ .ClusterName }}-tikv.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-tikv-peer"
  - "{{ .ClusterName }}-tikv-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-tikv-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-tikv-peer"
  - "*.{{ .ClusterName }}-tikv-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-tikv-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-tidb-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-tidb-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-tidb"
  - "{{ .ClusterName }}-tidb.{{ .Namespace }}"
  - "{{ .ClusterName }}-tidb.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-tidb-peer"
  - "{{ .ClusterName }}-tidb-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-tidb-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-tidb-peer"
  - "*.{{ .ClusterName }}-tidb-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-tidb-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-cluster-client-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-cluster-client-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - client auth
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-pump-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-pump-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "*.{{ .ClusterName }}-pump"
  - "*.{{ .ClusterName }}-pump.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-pump.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-drainer-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-drainer-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "*.{{ .ClusterName }}-{{ .ClusterName }}-drainer"
  - "*.{{ .ClusterName }}-{{ .ClusterName }}-drainer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-{{ .ClusterName }}-drainer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-tiflash-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-tiflash-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-tiflash"
  - "{{ .ClusterName }}-tiflash.{{ .Namespace }}"
  - "{{ .ClusterName }}-tiflash.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-tiflash-peer"
  - "{{ .ClusterName }}-tiflash-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-tiflash-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-tiflash-peer"
  - "*.{{ .ClusterName }}-tiflash-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-tiflash-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
`

var tidbClientCertificateTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .ClusterName }}-{{ .Component }}-tls
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-{{ .Component }}-tls
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
    - PingCAP
  commonName: "TiDB Client"
  usages:
    - client auth
  issuerRef:
    name: {{ .ClusterRef }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
`

type tcTmplMeta struct {
	Namespace   string
	ClusterName string
	ClusterRef  string
}

type tcCliTmplMeta struct {
	tcTmplMeta
	Component string
}

func installTiDBIssuer(ns, tcName string) error {
	return installCert(tidbIssuerTmpl, tcTmplMeta{ns, tcName, tcName})
}

func installTiDBCertificates(ns, tcName string) error {
	return installCert(tidbCertificatesTmpl, tcTmplMeta{ns, tcName, tcName})
}

func installHeterogeneousTiDBCertificates(ns, tcName string, clusterRef string) error {
	return installCert(tidbCertificatesTmpl, tcTmplMeta{ns, tcName, clusterRef})
}

func installTiDBComponentsCertificates(ns, tcName string) error {
	return installCert(tidbComponentsCertificatesTmpl, tcTmplMeta{ns, tcName, tcName})
}

func installHeterogeneousTiDBComponentsCertificates(ns, tcName string, clusterRef string) error {
	return installCert(tidbComponentsCertificatesTmpl, tcTmplMeta{ns, tcName, clusterRef})
}

func installTiDBInitializerCertificates(ns, tcName string) error {
	return installCert(tidbClientCertificateTmpl, tcCliTmplMeta{tcTmplMeta{ns, tcName, tcName}, "initializer"})
}

func installPDDashboardCertificates(ns, tcName string) error {
	return installCert(tidbClientCertificateTmpl, tcCliTmplMeta{tcTmplMeta{ns, tcName, tcName}, "dashboard"})
}

func installBackupCertificates(ns, tcName string) error {
	return installCert(tidbClientCertificateTmpl, tcCliTmplMeta{tcTmplMeta{ns, tcName, tcName}, "backup"})
}

func installRestoreCertificates(ns, tcName string) error {
	return installCert(tidbClientCertificateTmpl, tcCliTmplMeta{tcTmplMeta{ns, tcName, tcName}, "restore"})
}

func installCert(tmplStr string, tp interface{}) error {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error when parsing template: %v", err)
	}
	err = tmpl.Execute(&buf, tp)
	if err != nil {
		return fmt.Errorf("error when executing template: %v", err)
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "tls-")
	if err != nil {
		return err
	}
	_, err = tmpFile.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if data, err := exec.Command("sh", "-c", fmt.Sprintf("kubectl apply -f %s", tmpFile.Name())).CombinedOutput(); err != nil {
		framework.Logf("failed to create certificate: %s, %v", string(data), err)
		return err
	}

	return nil
}

func tidbIsTLSEnabled(fw portforward.PortForward, c clientset.Interface, ns, tcName, passwd string) wait.ConditionFunc {
	return func() (bool, error) {
		db, cancel, err := connectToTiDBWithTLSSupport(fw, c, ns, tcName, passwd, true)
		if err != nil {
			return false, nil
		}
		defer db.Close()
		defer cancel()

		rows, err := db.Query("SHOW STATUS")
		if err != nil {
			return false, err
		}
		var name, value string
		for rows.Next() {
			err := rows.Scan(&name, &value)
			if err != nil {
				return false, err
			}

			if name == "Ssl_cipher" {
				if value == "" {
					return true, fmt.Errorf("the connection to tidb server is not ssl %s/%s", ns, tcName)
				}

				framework.Logf("The connection to TiDB Server is TLS enabled.")
				return true, nil
			}
		}

		return true, fmt.Errorf("can't find Ssl_cipher in status %s/%s", ns, tcName)
	}
}

func insertIntoDataToSourceDB(fw portforward.PortForward, c clientset.Interface, ns, tcName, passwd string, tlsEnabled bool) wait.ConditionFunc {
	return func() (bool, error) {
		db, cancel, err := connectToTiDBWithTLSSupport(fw, c, ns, tcName, passwd, tlsEnabled)
		if err != nil {
			framework.Logf("failed to connect to source db: %v", err)
			return false, nil
		}
		defer db.Close()
		defer cancel()

		res, err := db.Exec("CREATE TABLE test.city (name VARCHAR(64) PRIMARY KEY)")
		if err != nil {
			framework.Logf("can't create table in source db: %v, %v", res, err)
			return false, nil
		}

		res, err = db.Exec("INSERT INTO test.city (name) VALUES (\"beijing\")")
		if err != nil {
			framework.Logf("can't insert into table tls in source db: %v, %v", res, err)
			return false, nil
		}

		return true, nil
	}
}

func dataInClusterIsCorrect(fw portforward.PortForward, c clientset.Interface, ns, tcName, passwd string, tlsEnabled bool) wait.ConditionFunc {
	return func() (bool, error) {
		db, cancel, err := connectToTiDBWithTLSSupport(fw, c, ns, tcName, passwd, tlsEnabled)
		if err != nil {
			framework.Logf("can't connect to %s/%s, %v", ns, tcName, err)
			return false, nil
		}
		defer db.Close()
		defer cancel()

		row := db.QueryRow("SELECT name from test.city limit 1")
		var name string

		err = row.Scan(&name)
		if err != nil {
			framework.Logf("can't scan from %s/%s, %v", ns, tcName, err)
			return false, nil
		}

		framework.Logf("TABLE test.city name = %s", name)
		if name == "beijing" {
			return true, nil
		}

		return false, nil
	}
}

func connectToTiDBWithTLSSupport(fw portforward.PortForward, c clientset.Interface, ns, tcName, passwd string, tlsEnabled bool) (*sql.DB, context.CancelFunc, error) {
	var tlsParams string

	localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, fmt.Sprintf("svc/%s", controller.TiDBMemberName(tcName)), 4000)
	if err != nil {
		return nil, nil, err
	}

	if tlsEnabled {
		tlsKey := "tidb-server-tls"
		secretName := util.TiDBClientTLSSecretName(tcName)
		secret, err := c.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}

		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(secret.Data[v1.ServiceAccountRootCAKey])

		clientCert, certExists := secret.Data[v1.TLSCertKey]
		clientKey, keyExists := secret.Data[v1.TLSPrivateKeyKey]
		if !certExists || !keyExists {
			return nil, nil, fmt.Errorf("cert or key does not exist in secret %s/%s", ns, secretName)
		}

		tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", ns, secretName, err)
		}
		err = mysql.RegisterTLSConfig(tlsKey, &tls.Config{
			RootCAs:            rootCAs,
			Certificates:       []tls.Certificate{tlsCert},
			InsecureSkipVerify: true,
		})
		if err != nil {
			return nil, nil, err
		}

		tlsParams = fmt.Sprintf("?tls=%s", tlsKey)
	}

	db, err := sql.Open("mysql",
		fmt.Sprintf("root:%s@(%s:%d)/test%s", passwd, localHost, localPort, tlsParams))
	if err != nil {
		return nil, nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, nil, err
	}

	return db, cancel, err
}
