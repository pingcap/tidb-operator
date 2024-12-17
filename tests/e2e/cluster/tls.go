// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/pingcap/tidb-operator/tests/e2e/utils/k8s"
)

var tidbIssuerTmpl = `
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: {{ .ClusterName }}-selfsigned-ca-issuer
  namespace: {{ .Namespace }}
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
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
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: {{ .ClusterName }}-tidb-issuer
  namespace: {{ .Namespace }}
spec:
  ca:
    secretName: {{ .ClusterName }}-ca-secret
`

var tidbCertificatesTmpl = `
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .ClusterName }}-{{ .TiDBGroupName}}-server-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-{{ .TiDBGroupName}}-server-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB Server"
  usages:
    - server auth
  dnsNames:
    - "{{ .ClusterName }}-{{ .TiDBGroupName}}"
    - "{{ .ClusterName }}-{{ .TiDBGroupName}}.{{ .Namespace }}"
    - "{{ .ClusterName }}-{{ .TiDBGroupName}}.{{ .Namespace }}.svc"
    - "*.{{ .ClusterName }}-{{ .TiDBGroupName}}"
    - "*.{{ .ClusterName }}-{{ .TiDBGroupName}}.{{ .Namespace }}"
    - "*.{{ .ClusterName }}-{{ .TiDBGroupName}}.{{ .Namespace }}.svc"
  ipAddresses:
    - 127.0.0.1
    - ::1
  issuerRef:
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .ClusterName }}-{{ .TiDBGroupName}}-client-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-{{ .TiDBGroupName}}-client-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB Client"
  usages:
    - client auth
  issuerRef:
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
`

var tidbComponentsCertificatesTmpl = `
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .ClusterName }}-{{ .PDGroupName }}-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-{{ .PDGroupName }}-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-{{ .PDGroupName }}"
  - "{{ .ClusterName }}-{{ .PDGroupName }}.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .PDGroupName }}.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-{{ .PDGroupName }}-peer"
  - "{{ .ClusterName }}-{{ .PDGroupName }}-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .PDGroupName }}-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-{{ .PDGroupName }}-peer"
  - "*.{{ .ClusterName }}-{{ .PDGroupName }}-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-{{ .PDGroupName }}-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .ClusterName }}-{{ .TiKVGroupName }}-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-{{ .TiKVGroupName }}-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-{{ .TiKVGroupName }}"
  - "{{ .ClusterName }}-{{ .TiKVGroupName }}.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .TiKVGroupName }}.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-{{ .TiKVGroupName }}-peer"
  - "{{ .ClusterName }}-{{ .TiKVGroupName }}-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .TiKVGroupName }}-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-{{ .TiKVGroupName }}-peer"
  - "*.{{ .ClusterName }}-{{ .TiKVGroupName }}-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-{{ .TiKVGroupName }}-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .ClusterName }}-{{ .TiDBGroupName }}-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-{{ .TiDBGroupName }}-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-{{ .TiDBGroupName }}"
  - "{{ .ClusterName }}-{{ .TiDBGroupName }}.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .TiDBGroupName }}.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-{{ .TiDBGroupName }}-peer"
  - "{{ .ClusterName }}-{{ .TiDBGroupName }}-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .TiDBGroupName }}-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-{{ .TiDBGroupName }}-peer"
  - "*.{{ .ClusterName }}-{{ .TiDBGroupName }}-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-{{ .TiDBGroupName }}-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .ClusterName }}-{{ .TiFlashGroupName }}-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-{{ .TiFlashGroupName }}-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
  - "{{ .ClusterName }}-{{ .TiFlashGroupName }}"
  - "{{ .ClusterName }}-{{ .TiFlashGroupName }}.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .TiFlashGroupName }}.{{ .Namespace }}.svc"
  - "{{ .ClusterName }}-{{ .TiFlashGroupName }}-peer"
  - "{{ .ClusterName }}-{{ .TiFlashGroupName }}-peer.{{ .Namespace }}"
  - "{{ .ClusterName }}-{{ .TiFlashGroupName }}-peer.{{ .Namespace }}.svc"
  - "*.{{ .ClusterName }}-{{ .TiFlashGroupName }}-peer"
  - "*.{{ .ClusterName }}-{{ .TiFlashGroupName }}-peer.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-{{ .TiFlashGroupName }}-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .ClusterName }}-cluster-client-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .ClusterName }}-cluster-client-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB"
  usages:
    - client auth
  issuerRef:
    name: {{ .ClusterName }}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
`

type tcTmplMeta struct {
	Namespace        string
	ClusterName      string
	PDGroupName      string
	TiKVGroupName    string
	TiDBGroupName    string
	TiFlashGroupName string
}

func installTiDBIssuer(ctx context.Context, yamlApplier *k8s.YAMLApplier, ns, clusterName string) error {
	return installCert(ctx, yamlApplier, tidbIssuerTmpl, tcTmplMeta{Namespace: ns, ClusterName: clusterName})
}

func installTiDBCertificates(ctx context.Context, yamlApplier *k8s.YAMLApplier, ns, clusterName, tidbGroupName string) error {
	return installCert(ctx, yamlApplier, tidbCertificatesTmpl, tcTmplMeta{
		Namespace: ns, ClusterName: clusterName, TiDBGroupName: tidbGroupName})
}

func installTiDBComponentsCertificates(ctx context.Context, yamlApplier *k8s.YAMLApplier, ns, clusterName string,
	pdGroupName, tikvGroupName, tidbGroupName, tiFlashGroupName string) error {
	return installCert(ctx, yamlApplier, tidbComponentsCertificatesTmpl, tcTmplMeta{
		Namespace: ns, ClusterName: clusterName,
		PDGroupName: pdGroupName, TiKVGroupName: tikvGroupName, TiDBGroupName: tidbGroupName, TiFlashGroupName: tiFlashGroupName})
}

func installCert(ctx context.Context, yamlApplier *k8s.YAMLApplier, tmplStr string, tp any) error {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error when parsing template: %w", err)
	}
	err = tmpl.Execute(&buf, tp)
	if err != nil {
		return fmt.Errorf("error when executing template: %w", err)
	}

	return yamlApplier.Apply(ctx, &buf)
}
