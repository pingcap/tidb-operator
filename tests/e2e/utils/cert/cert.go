package cert

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/pingcap/tidb-operator/pkg/client"
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
  name: {{ .TiDBGroupName}}-tidb-server-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .TiDBGroupName}}-tidb-server-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "TiDB Server"
  usages:
    - server auth
  dnsNames:
    - "{{ .TiDBGroupName}}-tidb"
    - "{{ .TiDBGroupName}}-tidb.{{ .Namespace }}"
    - "{{ .TiDBGroupName}}-tidb.{{ .Namespace }}.svc"
    - "*.{{ .TiDBGroupName}}-tidb"
    - "*.{{ .TiDBGroupName}}-tidb.{{ .Namespace }}"
    - "*.{{ .TiDBGroupName}}-tidb.{{ .Namespace }}.svc"
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
  name: {{ .TiDBGroupName}}-tidb-client-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .TiDBGroupName}}-tidb-client-secret
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
  name: {{ .PDGroupName }}-pd-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .PDGroupName }}-pd-cluster-secret
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
  - "{{ .PDGroupName }}-pd"
  - "{{ .PDGroupName }}-pd.{{ .Namespace }}"
  - "{{ .PDGroupName }}-pd.{{ .Namespace }}.svc"
  - "{{ .PDGroupName }}-pd-peer"
  - "{{ .PDGroupName }}-pd-peer.{{ .Namespace }}"
  - "{{ .PDGroupName }}-pd-peer.{{ .Namespace }}.svc"
  - "*.{{ .PDGroupName }}-pd-peer"
  - "*.{{ .PDGroupName }}-pd-peer.{{ .Namespace }}"
  - "*.{{ .PDGroupName }}-pd-peer.{{ .Namespace }}.svc"
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
  name: {{ .TiKVGroupName }}-tikv-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .TiKVGroupName }}-tikv-cluster-secret
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
  - "{{ .TiKVGroupName }}-tikv"
  - "{{ .TiKVGroupName }}-tikv.{{ .Namespace }}"
  - "{{ .TiKVGroupName }}-tikv.{{ .Namespace }}.svc"
  - "{{ .TiKVGroupName }}-tikv-peer"
  - "{{ .TiKVGroupName }}-tikv-peer.{{ .Namespace }}"
  - "{{ .TiKVGroupName }}-tikv-peer.{{ .Namespace }}.svc"
  - "*.{{ .TiKVGroupName }}-tikv-peer"
  - "*.{{ .TiKVGroupName }}-tikv-peer.{{ .Namespace }}"
  - "*.{{ .TiKVGroupName }}-tikv-peer.{{ .Namespace }}.svc"
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
  name: {{ .TiDBGroupName }}-tidb-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .TiDBGroupName }}-tidb-cluster-secret
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
  - "{{ .TiDBGroupName }}-tidb"
  - "{{ .TiDBGroupName }}-tidb.{{ .Namespace }}"
  - "{{ .TiDBGroupName }}-tidb.{{ .Namespace }}.svc"
  - "{{ .TiDBGroupName }}-tidb-peer"
  - "{{ .TiDBGroupName }}-tidb-peer.{{ .Namespace }}"
  - "{{ .TiDBGroupName }}-tidb-peer.{{ .Namespace }}.svc"
  - "*.{{ .TiDBGroupName }}-tidb-peer"
  - "*.{{ .TiDBGroupName }}-tidb-peer.{{ .Namespace }}"
  - "*.{{ .TiDBGroupName }}-tidb-peer.{{ .Namespace }}.svc"
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
  name: {{ .TiFlashGroupName }}-tiflash-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .TiFlashGroupName }}-tiflash-cluster-secret
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
  - "{{ .TiFlashGroupName }}-tiflash"
  - "{{ .TiFlashGroupName }}-tiflash.{{ .Namespace }}"
  - "{{ .TiFlashGroupName }}-tiflash.{{ .Namespace }}.svc"
  - "{{ .TiFlashGroupName }}-tiflash-peer"
  - "{{ .TiFlashGroupName }}-tiflash-peer.{{ .Namespace }}"
  - "{{ .TiFlashGroupName }}-tiflash-peer.{{ .Namespace }}.svc"
  - "*.{{ .TiFlashGroupName }}-tiflash-peer"
  - "*.{{ .TiFlashGroupName }}-tiflash-peer.{{ .Namespace }}"
  - "*.{{ .TiFlashGroupName }}-tiflash-peer.{{ .Namespace }}.svc"
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
  name: {{ .TiCDCGroupName }}-ticdc-cluster-secret
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .TiCDCGroupName }}-ticdc-cluster-secret
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
  - "{{ .TiCDCGroupName }}-ticdc-peer"
  - "{{ .TiCDCGroupName }}-ticdc-peer.{{ .Namespace }}"
  - "{{ .TiCDCGroupName }}-ticdc-peer.{{ .Namespace }}.svc"
  - "*.{{ .TiCDCGroupName }}-ticdc-peer"
  - "*.{{ .TiCDCGroupName }}-ticdc-peer.{{ .Namespace }}"
  - "*.{{ .TiCDCGroupName }}-ticdc-peer.{{ .Namespace }}.svc"
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
	TiCDCGroupName   string
}

func InstallTiDBIssuer(ctx context.Context, c client.Client, ns, clusterName string) error {
	return installCert(ctx, c, tidbIssuerTmpl, tcTmplMeta{Namespace: ns, ClusterName: clusterName})
}

func InstallTiDBCertificates(ctx context.Context, c client.Client, ns, clusterName, tidbGroupName string) error {
	return installCert(ctx, c, tidbCertificatesTmpl, tcTmplMeta{
		Namespace: ns, ClusterName: clusterName, TiDBGroupName: tidbGroupName,
	})
}

func InstallTiDBComponentsCertificates(ctx context.Context, c client.Client, ns, clusterName string,
	pdGroupName, tikvGroupName, tidbGroupName, tiFlashGroupName, tiCDCGroupName string,
) error {
	return installCert(ctx, c, tidbComponentsCertificatesTmpl, tcTmplMeta{
		Namespace: ns, ClusterName: clusterName,
		PDGroupName: pdGroupName, TiKVGroupName: tikvGroupName,
		TiDBGroupName: tidbGroupName, TiFlashGroupName: tiFlashGroupName, TiCDCGroupName: tiCDCGroupName,
	})
}

func installCert(ctx context.Context, c client.Client, tmplStr string, tp any) error {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error when parsing template: %w", err)
	}
	err = tmpl.Execute(&buf, tp)
	if err != nil {
		return fmt.Errorf("error when executing template: %w", err)
	}

	objs, err := DecodeYAML(&buf)
	if err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	for _, obj := range objs {
		if err := c.Create(ctx, obj); err != nil {
			return fmt.Errorf("cannot create %s: %w", client.ObjectKeyFromObject(obj), err)
		}
	}

	return nil
}

func DecodeYAML(r io.Reader) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	//nolint:mnd // refactor to a constant if needed
	decoder := yaml.NewYAMLOrJSONDecoder(r, 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		objects = append(objects, obj)
	}
	return objects, nil
}
