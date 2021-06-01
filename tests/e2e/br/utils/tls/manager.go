// Copyright 2021 PingCAP, Inc.
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

package tls

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	yamlutil "github.com/pingcap/tidb-operator/tests/e2e/br/utils/yaml"
)

var (
	issuerTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: {{ .Name }}-selfsigned-ca-issuer
  namespace: {{ .Namespace }}
spec:
  selfSigned: {}
`

	caTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .Name }}-ca
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .Name }}-ca-secret
  commonName: "TiDB CA"
  isCA: true
  issuerRef:
    name: {{ .Name }}-selfsigned-ca-issuer
    kind: Issuer
`

	certIssuerTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: {{ .Name }}-cert-issuer
  namespace: {{ .Namespace }}
spec:
  ca:
    secretName: {{ .Name }}-ca-secret
`

	certTmpl = `
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .SecretName }}
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "{{ .CN }}"
  {{- if .Usages }}
  usages:
  {{- range .Usages }}
  - "{{ . }}"
  {{- end }}
  {{- end }}

  {{- if .DNSNames }}
  dnsNames:
  {{- range .DNSNames }}
  - "{{ . }}"
  {{- end }}
  {{- end }}

  {{- if .IPAddresses }}
  ipAddresses:
  {{- range .IPAddresses }}
  - "{{ . }}"
  {{- end }}
  {{- end }}

  issuerRef:
    name: {{ .Issuer }}
    kind: Issuer
    group: cert-manager.io
`
)

type Meta struct {
	Name      string
	Namespace string
}

type TLSCert struct {
	Meta
	SecretName  string
	CN          string
	Usages      []string
	DNSNames    []string
	IPAddresses []string
	Issuer      string
}

// Manager is defined for managing tls in e2e tests
type Manager interface {
	CreateTLSForTidbCluster(tc *v1alpha1.TidbCluster) error
}

func New(c yamlutil.Interface) Manager {
	return &manager{
		c: c,
	}
}

type manager struct {
	c yamlutil.Interface
}

func (m *manager) CreateTLSForTidbCluster(tc *v1alpha1.TidbCluster) error {
	if err := m.createCA(tc.Namespace, tc.Name); err != nil {
		return err
	}
	tidbSpec := tc.Spec.TiDB
	// create certs for outer tidb client
	if tidbSpec != nil && tidbSpec.TLSClient != nil && tidbSpec.TLSClient.Enabled {
		if err := m.createCert(tidbClientCert(tc)); err != nil {
			return err
		}
		if err := m.createCert(tidbServerCert(tc)); err != nil {
			return err
		}
	}
	// create certs for inner communication
	if tc.Spec.TLSCluster != nil || tc.Spec.TLSCluster.Enabled {
		// create client certs
		if err := m.createCert(clusterClientCert(tc)); err != nil {
			return err
		}

		// create components certs
		if tc.Spec.PD != nil {
			if err := m.createCert(pdClusterCert(tc)); err != nil {
				return err
			}
		}
		if tc.Spec.TiKV != nil {
			if err := m.createCert(tikvClusterCert(tc)); err != nil {
				return err
			}
		}
		if tc.Spec.TiDB != nil {
			if err := m.createCert(tidbClusterCert(tc)); err != nil {
				return err
			}
		}
		if tc.Spec.TiFlash != nil {
			if err := m.createCert(tiflashClusterCert(tc)); err != nil {
				return err
			}
		}
		if tc.Spec.Pump != nil {
			if err := m.createCert(pumpClusterCert(tc)); err != nil {
				return err
			}
		}
		if tc.Spec.TiCDC != nil {
			if err := m.createCert(ticdcClusterCert(tc)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *manager) createCA(ns, name string) error {
	meta := Meta{
		Name:      name,
		Namespace: ns,
	}
	if err := m.createFromTemplate(issuerTmpl, &meta); err != nil {
		return err
	}
	if err := m.createFromTemplate(caTmpl, &meta); err != nil {
		return err
	}
	if err := m.createFromTemplate(certIssuerTmpl, &meta); err != nil {
		return err
	}
	return nil
}

func (m *manager) createCert(cert *TLSCert) error {
	return m.createFromTemplate(certTmpl, cert)
}

func tidbClientCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.TiDBClientTLSSecretName(tc.Name)

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB Client",
		Usages:     []string{"client auth"},
		Issuer:     certIssuer(tc),
	}
}
func tidbServerCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.TiDBServerTLSSecretName(tc.Name)

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB Server",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			tc.Name + "-tidb",
			"*." + tc.Name + "-tidb",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"server auth"},
		Issuer: certIssuer(tc),
	}
}

func pdClusterCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterTLSSecretName(tc.Name, "pd")

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			tc.Name + "-pd",
			tc.Name + "-pd-peer",
			"*." + tc.Name + "-pd-peer",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"client auth", "server auth"},
		Issuer: certIssuer(tc),
	}
}

func tikvClusterCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterTLSSecretName(tc.Name, "tikv")

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			tc.Name + "-tikv",
			tc.Name + "-tikv-peer",
			"*." + tc.Name + "-tikv-peer",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"client auth", "server auth"},
		Issuer: certIssuer(tc),
	}
}

func tidbClusterCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterTLSSecretName(tc.Name, "tidb")

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			tc.Name + "-tidb",
			tc.Name + "-tidb-peer",
			"*." + tc.Name + "-tidb-peer",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"client auth", "server auth"},
		Issuer: certIssuer(tc),
	}
}

func tiflashClusterCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterTLSSecretName(tc.Name, "tiflash")

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			tc.Name + "-tiflash",
			tc.Name + "-tiflash-peer",
			"*." + tc.Name + "-tiflash-peer",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"client auth", "server auth"},
		Issuer: certIssuer(tc),
	}
}

func pumpClusterCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterTLSSecretName(tc.Name, "pump")

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			"*." + tc.Name + "-pump",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"client auth", "server auth"},
		Issuer: certIssuer(tc),
	}
}
func ticdcClusterCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterTLSSecretName(tc.Name, "ticdc")

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			tc.Name + "-ticdc",
			tc.Name + "-ticdc-peer",
			"*." + tc.Name + "-ticdc-peer",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"client auth", "server auth"},
		Issuer: certIssuer(tc),
	}
}

// nolint
// NOTE: it is not used now
func drainerClusterCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterTLSSecretName(tc.Name, "drainer")

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		DNSNames: dnsNamesExpand(tc.Namespace, []string{
			"*." + tc.Name + "-" + tc.Name + "-drainer",
		}),
		IPAddresses: []string{
			"127.0.0.1",
			"::1",
		},
		Usages: []string{"client auth", "server auth"},
		Issuer: certIssuer(tc),
	}
}

func clusterClientCert(tc *v1alpha1.TidbCluster) *TLSCert {
	secretName := util.ClusterClientTLSSecretName(tc.Name)

	return &TLSCert{
		Meta: Meta{
			Name:      secretName,
			Namespace: tc.Namespace,
		},
		SecretName: secretName,
		CN:         "TiDB",
		Usages:     []string{"client auth"},
		Issuer:     certIssuer(tc),
	}
}

func dnsNamesExpand(ns string, dnsNames []string) []string {
	expanded := []string{}
	for _, n := range dnsNames {
		expanded = append(expanded, n)
		expanded = append(expanded, n+"."+ns)
		expanded = append(expanded, n+"."+ns+".svc")
	}
	return expanded
}

func certIssuer(tc *v1alpha1.TidbCluster) string {
	return tc.Name + "-cert-issuer"
}

func (m *manager) createFromTemplate(tmpl string, args interface{}) error {
	var buf bytes.Buffer
	t, err := template.New("template").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("error when parsing template: %v", err)
	}
	if err := t.Execute(&buf, args); err != nil {
		return fmt.Errorf("error when executing template: %v", err)
	}
	if err := m.c.Create(buf.Bytes()); err != nil {
		return err
	}
	return nil
}
