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

package cert

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/yaml"
)

// MySQLClient returns ca and certKeyPair for a mysql client
// The input is ca and certKeyPair of the MySQL server
func MySQLClient(clientCA, serverCertKeyPair string) (string, string) {
	return clientCA + "-server", serverCertKeyPair + "-client"
}

// Factory is used to create all certs of a cluster
type Factory interface {
	Install(ctx context.Context, ns, cluster string) error
	Cleanup(ctx context.Context) error
}

func NewFactory(c client.Client) Factory {
	return &factory{
		c:       c,
		secrets: map[string]*Secret{},
	}
}

var selfSignedIssuer = `
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-ca-issuer
spec:
  selfSigned: {}
`

var caIssuerTmpl = `
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .Namespace }}-{{ .CAIssuer }}
  namespace: cert-manager
spec:
  secretName: {{ .Namespace }}-{{ .CAIssuer }}
  commonName: "{{ .CN }}"
  isCA: true
  issuerRef:
    name: selfsigned-ca-issuer
    kind: ClusterIssuer
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: {{ .Namespace }}-{{ .CAIssuer }}
spec:
  ca:
    secretName: {{ .Namespace }}-{{ .CAIssuer }}
`

type caIssuerData struct {
	Namespace string
	CAIssuer  string
	CN        string
}

var caTmpl = `
apiVersion: trust.cert-manager.io/v1alpha1
kind: Bundle
metadata:
  name: {{ .CA }}
spec:
  sources:
  - secret:
      name: "{{ .Namespace }}-{{ .CAIssuer }}"
      key: "tls.crt"
  target:
    secret:
      key: "ca.crt"
    namespaceSelector:
      kubernetes.io/metadata.name: {{ .Namespace }}
`

type caData struct {
	Namespace string
	CAIssuer  string
	CA        string
}

var certKeyPairTmpl = `
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .Cert }}
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .Cert }}
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "{{ .CN }}"
  usages:
    - server auth
    {{ if .ClientAuth }}- client auth{{ end }}
  dnsNames:
  - "{{ .GroupName }}-{{ .ComponentName }}"
  - "{{ .GroupName }}-{{ .ComponentName }}.{{ .Namespace }}"
  - "{{ .GroupName }}-{{ .ComponentName }}.{{ .Namespace }}.svc"
  - "{{ .GroupName }}-{{ .ComponentName }}-peer"
  - "{{ .GroupName }}-{{ .ComponentName }}-peer.{{ .Namespace }}"
  - "{{ .GroupName }}-{{ .ComponentName }}-peer.{{ .Namespace }}.svc"
  {{- if .ClusterSubdomain }}
  - "*.{{ .ClusterName }}-cluster"
  - "*.{{ .ClusterName }}-cluster.{{ .Namespace }}"
  - "*.{{ .ClusterName }}-cluster.{{ .Namespace }}.svc"
  {{- else }}
  - "*.{{ .GroupName }}-{{ .ComponentName }}-peer"
  - "*.{{ .GroupName }}-{{ .ComponentName }}-peer.{{ .Namespace }}"
  - "*.{{ .GroupName }}-{{ .ComponentName }}-peer.{{ .Namespace }}.svc"
  {{- end }}
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .Namespace }}-{{ .CAIssuer }}
    kind: ClusterIssuer
    group: cert-manager.io
`

type certKeyPairData struct {
	Namespace        string
	CAIssuer         string
	CN               string
	Cert             string
	GroupName        string
	ComponentName    string
	ClusterName      string
	ClientAuth       bool
	ClusterSubdomain bool
}

var clientCertKeyPairTmpl = `
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .Cert }}
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .Cert }}
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  subject:
    organizations:
      - PingCAP
  commonName: "{{ .CN }}"
  usages:
    - client auth
  issuerRef:
    name: {{ .Namespace }}-{{ .CAIssuer }}
    kind: ClusterIssuer
    group: cert-manager.io
`

type clientCertKeyPairData struct {
	Namespace string
	CAIssuer  string
	CN        string
	Cert      string
}

func registerTiDBMySQLCerts(ctx context.Context, f *factory, ns, cluster string) error {
	c, err := apicall.GetClusterByKey(ctx, f.c, ns, cluster)
	if err != nil {
		return err
	}
	fg := features.NewFromFeatures(coreutil.EnabledFeatures(c))

	gs, err := apicall.ListGroups[scope.TiDBGroup](ctx, f.c, ns, cluster)
	if err != nil {
		return err
	}
	for _, g := range gs {
		ca := coreutil.TiDBGroupMySQLCASecretName(g)
		certKeyPair := coreutil.TiDBGroupMySQLCertKeyPairSecretName(g)
		s, err := newCertKeyPair(typeTiDBMySQLServer, ns, certKeyPair, cluster, g.GetName(), scope.Component[scope.TiDBGroup](), false, fg.Enabled(metav1alpha1.ClusterSubdomain))
		if err != nil {
			return err
		}
		if err := f.RegisterSecret(s); err != nil {
			return err
		}
		if ca != certKeyPair {
			as, err := newCA(typeTiDBMySQLClient, ns, ca, cluster)
			if err != nil {
				return err
			}
			if err := f.RegisterSecret(as); err != nil {
				return err
			}
		}
		clientCA, clientCertKeyPair := MySQLClient(ca, certKeyPair)
		cs, err := newClientCertKeyPair(typeTiDBMySQLClient, ns, clientCertKeyPair, cluster)
		if err != nil {
			return err
		}
		if err := f.RegisterSecret(cs); err != nil {
			return err
		}
		if clientCA != clientCertKeyPair {
			cas, err := newCA(typeTiDBMySQLServer, ns, clientCA, cluster)
			if err != nil {
				return err
			}
			if err := f.RegisterSecret(cas); err != nil {
				return err
			}
		}
	}

	return nil
}

func registerCertsForComponents[
	S scope.GroupList[F, T, L],
	F client.Object,
	T runtime.Group,
	L client.ObjectList,
](ctx context.Context, f *factory, ns, cluster string) error {
	c, err := apicall.GetClusterByKey(ctx, f.c, ns, cluster)
	if err != nil {
		return err
	}
	fg := features.NewFromFeatures(coreutil.EnabledFeatures(c))

	gs, err := apicall.ListGroups[S](ctx, f.c, ns, cluster)
	if err != nil {
		return err
	}
	for _, g := range gs {
		ca := coreutil.ClusterCASecretName[S](g)
		certKeyPair := coreutil.ClusterCertKeyPairSecretName[S](g)
		s, err := newCertKeyPair(typeCluster, ns, certKeyPair, cluster, g.GetName(), scope.Component[S](), true, fg.Enabled(metav1alpha1.ClusterSubdomain))
		if err != nil {
			return err
		}
		if err := f.RegisterSecret(s); err != nil {
			return err
		}
		if ca != certKeyPair {
			as, err := newCA(typeCluster, ns, ca, cluster)
			if err != nil {
				return err
			}
			if err := f.RegisterSecret(as); err != nil {
				return err
			}
		}
		clientCA := coreutil.ClientCASecretName[S](g)
		clientCertKeyPair := coreutil.ClientCertKeyPairSecretName[S](g)
		cs, err := newClientCertKeyPair(typeCluster, ns, clientCertKeyPair, cluster)
		if err != nil {
			return err
		}
		if err := f.RegisterSecret(cs); err != nil {
			return err
		}
		if clientCA != clientCertKeyPair {
			acs, err := newCA(typeCluster, ns, clientCA, cluster)
			if err != nil {
				return err
			}
			if err := f.RegisterSecret(acs); err != nil {
				return err
			}
		}
	}

	return nil
}

type factory struct {
	secrets map[string]*Secret
	c       client.Client
}

func (f *factory) registerCerts(ctx context.Context, ns, cluster string) error {
	if err := f.registerCAIssuer(ns, cluster); err != nil {
		return err
	}
	fs := []func(ctx context.Context, f *factory, ns, cluster string) error{
		registerCertsForComponents[scope.PDGroup],
		registerCertsForComponents[scope.TSOGroup],
		registerCertsForComponents[scope.SchedulerGroup],
		registerCertsForComponents[scope.TiDBGroup],
		registerCertsForComponents[scope.TiKVGroup],
		registerCertsForComponents[scope.TiFlashGroup],
		registerCertsForComponents[scope.TiCDCGroup],
		registerCertsForComponents[scope.TiProxyGroup],
		registerTiDBMySQLCerts,
	}

	for _, fn := range fs {
		if err := fn(ctx, f, ns, cluster); err != nil {
			return err
		}
	}

	return nil
}

func (f *factory) Install(ctx context.Context, ns, cluster string) error {
	// ensure selfSignedIssuer exists
	objs, err := newCertObjs(selfSignedIssuer, nil)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if err := f.c.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			if err := f.c.Create(ctx, obj); err != nil {
				if !errors.IsAlreadyExists(err) {
					return err
				}
			}
		}
	}

	if err := f.registerCerts(ctx, ns, cluster); err != nil {
		return err
	}

	for _, s := range f.secrets {
		for _, obj := range s.Objs {
			if err := f.c.Create(ctx, obj); err != nil {
				return err
			}
		}
	}

	return nil
}

func (f *factory) Cleanup(ctx context.Context) error {
	for _, s := range f.secrets {
		for _, obj := range s.Objs {
			if err := f.c.Delete(ctx, obj); err != nil {
				return err
			}
		}
	}

	return nil
}

func (f *factory) RegisterSecret(s *Secret) error {
	found, ok := f.secrets[s.Name]
	if !ok {
		f.secrets[s.Name] = s
		return nil
	}
	if found.Equal(s) {
		return nil
	}
	return fmt.Errorf("secret is conflict: %v, %v", s.Keys, found.Keys)
}

func (f *factory) registerCAIssuer(ns, cluster string) error {
	allTypes := []string{
		typeCluster,
		typeTiDBMySQLServer,
		typeTiDBMySQLClient,
	}
	for _, typ := range allTypes {
		s, err := newCAIssuer(typ, ns, cluster)
		if err != nil {
			return err
		}
		if err := f.RegisterSecret(s); err != nil {
			return err
		}
	}

	return nil
}

type Secret struct {
	Name string
	Keys []string
	Objs []*unstructured.Unstructured
}

const (
	typeCluster         = "cluster"
	typeTiDBMySQLServer = "tidb-mysql"
	typeTiDBMySQLClient = "tidb-mysql-client"
)

func (c *Secret) Equal(ac *Secret) bool {
	return reflect.DeepEqual(c.Keys, ac.Keys)
}

func newCAIssuerName(cluster, typ string) string {
	return cluster + "-" + typ + managedSuffix
}

func newCAIssuer(typ, ns, cluster string) (*Secret, error) {
	name := newCAIssuerName(cluster, typ)
	objs, err := newCertObjs(caIssuerTmpl, &caIssuerData{
		Namespace: ns,
		CAIssuer:  name,
		CN:        "PingCAP",
	})
	if err != nil {
		return nil, err
	}

	return &Secret{
		Name: name,
		Keys: []string{
			name,
			"issuer",
		},
		Objs: objs,
	}, nil
}

const managedSuffix = "-managed"

func newCertKeyPair(typ, ns, certKeyPair, cluster, groupName, component string, clientAuth bool, enableClusterSubdomain bool) (*Secret, error) {
	if strings.HasSuffix(certKeyPair, managedSuffix) {
		return nil, fmt.Errorf("secret with '%s' suffix is used", managedSuffix)
	}
	objs, err := newCertObjs(certKeyPairTmpl, &certKeyPairData{
		Namespace:        ns,
		CAIssuer:         newCAIssuerName(cluster, typ),
		Cert:             certKeyPair,
		ClusterName:      cluster,
		GroupName:        groupName,
		ComponentName:    component,
		ClientAuth:       clientAuth,
		CN:               "PingCAP",
		ClusterSubdomain: enableClusterSubdomain,
	})
	if err != nil {
		return nil, err
	}

	return &Secret{
		Name: certKeyPair,
		Keys: []string{
			certKeyPair,
			"internalCertKeyPair",
			groupName,
			component,
		},
		Objs: objs,
	}, nil
}

func newCA(typ, ns, ca, cluster string) (*Secret, error) {
	if strings.HasSuffix(ca, managedSuffix) {
		return nil, fmt.Errorf("secret with '%s' suffix is used", managedSuffix)
	}
	objs, err := newCertObjs(caTmpl, &caData{
		Namespace: ns,
		CAIssuer:  newCAIssuerName(cluster, typ),
		CA:        ca,
	})
	if err != nil {
		return nil, err
	}

	return &Secret{
		Name: ca,
		Keys: []string{
			ca,
			"ca",
		},
		Objs: objs,
	}, nil
}

func newClientCertKeyPair(typ, ns, certKeyPair, cluster string) (*Secret, error) {
	if strings.HasSuffix(certKeyPair, managedSuffix) {
		return nil, fmt.Errorf("secret with '%s' suffix is used", managedSuffix)
	}
	objs, err := newCertObjs(clientCertKeyPairTmpl, &clientCertKeyPairData{
		Namespace: ns,
		CAIssuer:  newCAIssuerName(cluster, typ),
		Cert:      certKeyPair,
		CN:        "PingCAP",
	})
	if err != nil {
		return nil, err
	}

	return &Secret{
		Name: certKeyPair,
		Keys: []string{
			certKeyPair,
			"internalClientCertKeyPair",
		},
		Objs: objs,
	}, nil
}

func newCertObjs(tmplStr string, tp any) ([]*unstructured.Unstructured, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("error when parsing template: %w", err)
	}
	err = tmpl.Execute(&buf, tp)
	if err != nil {
		return nil, fmt.Errorf("error when executing template: %w", err)
	}

	objs, err := yaml.DecodeYAML(&buf)
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	return objs, nil
}
