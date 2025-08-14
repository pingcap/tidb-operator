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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/pingcap/errors"

	"github.com/pingcap/tidb-operator/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/yaml"
)

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
      name: "{{ .Namespace}}-{{ .CAIssuer }}"
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
  - "{{ .Subdomain }}-{{ .ComponentName }}"
  - "{{ .Subdomain }}-{{ .ComponentName }}.{{ .Namespace }}"
  - "{{ .Subdomain }}-{{ .ComponentName }}.{{ .Namespace }}.svc"
  - "{{ .Subdomain }}-{{ .ComponentName }}-peer"
  - "{{ .Subdomain }}-{{ .ComponentName }}-peer.{{ .Namespace }}"
  - "{{ .Subdomain }}-{{ .ComponentName }}-peer.{{ .Namespace }}.svc"
  - "*.{{ .Subdomain }}-{{ .ComponentName }}-peer"
  - "*.{{ .Subdomain }}-{{ .ComponentName }}-peer.{{ .Namespace }}"
  - "*.{{ .Subdomain }}-{{ .ComponentName }}-peer.{{ .Namespace }}.svc"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: {{ .Namespace }}-{{ .CAIssuer }}
    kind: ClusterIssuer
    group: cert-manager.io
`

type certKeyPairData struct {
	Namespace     string
	CAIssuer      string
	CN            string
	Cert          string
	Subdomain     string
	ComponentName string
	ClientAuth    bool
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

func registerInternalCertsForComponents[
	S scope.GroupList[F, T, L],
	F client.Object,
	T runtime.Group,
	L client.ObjectList,
](ctx context.Context, f *factory, ns, cluster string) error {
	gs, err := apicall.ListGroups[S](ctx, f.c, ns, cluster)
	if err != nil {
		return err
	}
	for _, g := range gs {
		ca := coreutil.ClusterCASecretName[S](g)
		certKeyPair := coreutil.ClusterCertKeyPairSecretName[S](g)
		s, err := newInternalCertKeyPair(ns, certKeyPair, cluster, g.GetName(), scope.Component[S]())
		if err != nil {
			return err
		}
		if err := f.RegisterSecret(s); err != nil {
			return err
		}
		if ca != certKeyPair {
			s, err := newInternalCA(ns, ca, cluster)
			if err != nil {
				return err
			}
			if err := f.RegisterSecret(s); err != nil {
				return err
			}
		}
		clientCA := coreutil.ClientCASecretName[S](g)
		clientCertKeyPair := coreutil.ClientCertKeyPairSecretName[S](g)
		cs, err := newInternalClientCertKeyPair(ns, clientCertKeyPair, cluster)
		if err != nil {
			return err
		}
		if err := f.RegisterSecret(cs); err != nil {
			return err
		}
		if clientCA != clientCertKeyPair {
			cs, err := newInternalCA(ns, ca, cluster)
			if err != nil {
				return err
			}
			if err := f.RegisterSecret(cs); err != nil {
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

func (f *factory) registerInternalCerts(ctx context.Context, ns, cluster string) error {
	fs := []func(ctx context.Context, f *factory, ns, cluster string) error{
		registerInternalCertsForComponents[scope.PDGroup],
		registerInternalCertsForComponents[scope.TSOGroup],
		registerInternalCertsForComponents[scope.SchedulerGroup],
		registerInternalCertsForComponents[scope.TiDBGroup],
		registerInternalCertsForComponents[scope.TiKVGroup],
		registerInternalCertsForComponents[scope.TiFlashGroup],
		registerInternalCertsForComponents[scope.TiCDCGroup],
		registerInternalCertsForComponents[scope.TiProxyGroup],
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

	// create ca issuer
	s, err := newInternalTLSCAIssuer(ns, cluster)
	if err != nil {
		return err
	}
	if err := f.RegisterSecret(s); err != nil {
		return err
	}

	if err := f.registerInternalCerts(ctx, ns, cluster); err != nil {
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

type Secret struct {
	Name string
	Keys []string
	Objs []*unstructured.Unstructured
}

func (c *Secret) Equal(ac *Secret) bool {
	return reflect.DeepEqual(c.Keys, ac.Keys)
}

func newInternalTLSCAIssuer(ns, cluster string) (*Secret, error) {
	name := cluster + managedSuffix
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

func newInternalCertKeyPair(ns, certKeyPair, cluster, subdomain, component string) (*Secret, error) {
	if strings.HasSuffix(certKeyPair, managedSuffix) {
		return nil, fmt.Errorf("secret with '%s' suffix is used", managedSuffix)
	}
	objs, err := newCertObjs(certKeyPairTmpl, &certKeyPairData{
		Namespace:     ns,
		CAIssuer:      cluster + managedSuffix,
		Cert:          certKeyPair,
		Subdomain:     subdomain,
		ComponentName: component,
		ClientAuth:    true,
		CN:            "PingCAP",
	})
	if err != nil {
		return nil, err
	}

	return &Secret{
		Name: certKeyPair,
		Keys: []string{
			certKeyPair,
			"internalCertKeyPair",
			subdomain,
			component,
		},
		Objs: objs,
	}, nil
}

func newInternalCA(ns, ca, cluster string) (*Secret, error) {
	if strings.HasSuffix(ca, managedSuffix) {
		return nil, fmt.Errorf("secret with '%s' suffix is used", managedSuffix)
	}
	objs, err := newCertObjs(caTmpl, &caData{
		Namespace: ns,
		CAIssuer:  cluster + managedSuffix,
		CA:        ca,
	})
	if err != nil {
		return nil, err
	}

	return &Secret{
		Name: ca,
		Keys: []string{
			ca,
			"internalCA",
		},
		Objs: objs,
	}, nil
}

func newInternalClientCertKeyPair(ns, certKeyPair, cluster string) (*Secret, error) {
	if strings.HasSuffix(certKeyPair, managedSuffix) {
		return nil, fmt.Errorf("secret with '%s' suffix is used", managedSuffix)
	}
	objs, err := newCertObjs(clientCertKeyPairTmpl, &clientCertKeyPairData{
		Namespace: ns,
		CAIssuer:  cluster + managedSuffix,
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
