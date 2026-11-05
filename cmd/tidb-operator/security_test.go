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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrlcli "sigs.k8s.io/controller-runtime/pkg/client"

	operatorscheme "github.com/pingcap/tidb-operator/v2/pkg/scheme"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/kubefeat"
)

type secretTransport func(*http.Request) (*http.Response, error)

func (f secretTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type deniedCache struct {
	ctrlcli.Reader
	reads int
}

func (c *deniedCache) Get(context.Context, ctrlcli.ObjectKey, ctrlcli.Object, ...ctrlcli.GetOption) error {
	c.reads++
	return errors.New("cache used")
}
func TestSecretsBypassCache(t *testing.T) {
	kubefeat.MustInitFeatureGates(&rest.Config{Host: "https://api.invalid", Transport: secretTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("# TYPE kubernetes_feature_enabled gauge\nkubernetes_feature_enabled{name=\"VolumeAttributesClass\",stage=\"BETA\"} 1\n"))}, nil
	})})
	cache := &deniedCache{}
	calls := 0
	transport := secretTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/openapi/v3" {
			paths := map[string]any{}
			for _, gv := range operatorscheme.DynamicGroupVersions() {
				path := "apis/" + gv.String()
				if gv.Group == "" {
					path = "api/" + gv.Version
				}
				paths[path] = map[string]string{"serverRelativeURL": "/openapi/mock"}
			}
			body, err := json.Marshal(map[string]any{"paths": paths})
			require.NoError(t, err)
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(body))}, nil
		}
		if r.URL.Path == "/openapi/mock" {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		calls++
		require.Equal(t, "/api/v1/namespaces/tenant/secrets/storage", r.URL.Path)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"apiVersion":"v1","kind":"Secret","metadata":{"namespace":"tenant","name":"storage"},"data":{"test":"dmFsdWU="}}`))}, nil
	})
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{corev1.SchemeGroupVersion})
	mapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)
	mapper.Add(corev1.SchemeGroupVersion.WithKind("ConfigMap"), meta.RESTScopeNamespace)
	cli, err := newOperatorClient(&rest.Config{Host: "https://api.invalid", Transport: transport}, ctrlcli.Options{Scheme: scheme, Mapper: mapper, Cache: &ctrlcli.CacheOptions{Reader: cache}})
	require.NoError(t, err)
	secret := &corev1.Secret{}
	require.NoError(t, cli.Get(context.Background(), ctrlcli.ObjectKey{Namespace: "tenant", Name: "storage"}, secret))
	require.Equal(t, []byte("value"), secret.Data["test"])
	require.Zero(t, cache.reads)
	require.Equal(t, 1, calls)
	require.ErrorContains(t, cli.Get(context.Background(), ctrlcli.ObjectKey{Namespace: "tenant", Name: "config"}, &corev1.ConfigMap{}), "cache used")
	require.Equal(t, 1, cache.reads)
}
func TestWatchNamespaces(t *testing.T) {
	scopes, err := parseWatchNamespaces("")
	require.NoError(t, err)
	require.Nil(t, scopes)
	scopes, err = parseWatchNamespaces("tenant-a, tenant-b,tenant-a")
	require.NoError(t, err)
	require.Len(t, scopes, 2)
	for _, value := range []string{"tenant,", "BadNamespace", "tenant/*", "*", " "} {
		_, err := parseWatchNamespaces(value)
		require.Error(t, err, value)
	}
}
