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
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
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
    name: {{ .ClusterName }}-selfsigned-ca-issuer
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
    name: {{ .ClusterName }}-tidb-issuer
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
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
`

type tidbClusterTmplMeta struct {
	Namespace   string
	ClusterName string
}

func installCertManager(cli clientset.Interface) error {
	cmd := "kubectl apply -f /cert-manager.yaml --validate=false"
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install cert-manager %s %v", string(data), err)
	}

	err := e2epod.WaitForPodsRunningReady(cli, "cert-manager", 3, 0, 10*time.Minute, nil)
	if err != nil {
		return err
	}

	// It may take a minute or so for the TLS assets required for the webhook to function to be provisioned.
	time.Sleep(time.Minute)
	return nil
}

func deleteCertManager(cli clientset.Interface) error {
	cmd := "kubectl delete -f /cert-manager.yaml --ignore-not-found"
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete cert-manager %s %v", string(data), err)
	}

	return wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		podList, err := cli.CoreV1().Pods("cert-manager").List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		for _, pod := range podList.Items {
			err := e2epod.WaitForPodNotFoundInNamespace(cli, pod.Name, "cert-manager", 5*time.Minute)
			if err != nil {
				framework.Logf("failed to wait for pod cert-manager/%s disappear", pod.Name)
				return false, nil
			}
		}

		return true, nil
	})
}

func installTiDBIssuer(ns, tcName string) error {
	return installCert(tidbIssuerTmpl, ns, tcName)
}

func installTiDBCertificates(ns, tcName string) error {
	return installCert(tidbCertificatesTmpl, ns, tcName)
}

func installCert(tmplStr, ns, tcName string) error {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error when parsing template: %v", err)
	}
	err = tmpl.Execute(&buf, tidbClusterTmplMeta{ns, tcName})
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
		secretName := util.TiDBClientTLSSecretName(tcName)
		secret, err := c.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(secret.Data[v1.ServiceAccountRootCAKey])

		clientCert, certExists := secret.Data[v1.TLSCertKey]
		clientKey, keyExists := secret.Data[v1.TLSPrivateKeyKey]
		if !certExists || !keyExists {
			return false, fmt.Errorf("cert or key does not exist in secret %s/%s", ns, secretName)
		}

		tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
		if err != nil {
			return false, fmt.Errorf("unable to load certificates from secret %s/%s: %v", ns, secretName, err)
		}
		err = mysql.RegisterTLSConfig("tidb-server-tls", &tls.Config{
			RootCAs:            rootCAs,
			Certificates:       []tls.Certificate{tlsCert},
			InsecureSkipVerify: true,
		})
		if err != nil {
			return false, err
		}

		localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, fmt.Sprintf("svc/%s", controller.TiDBMemberName(tcName)), 4000)
		if err != nil {
			return false, err
		}
		defer cancel()

		db, err := sql.Open("mysql",
			fmt.Sprintf("root:%s@(%s:%d)/test?tls=tidb-server-tls", passwd, localHost, localPort))
		if err != nil {
			return false, err
		}

		defer db.Close()
		if err := db.Ping(); err != nil {
			return false, err
		}

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
