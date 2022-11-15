// Copyright 2022 PingCAP, Inc.
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

package controller

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/pingcap/TiProxy/lib/cli"
	"github.com/pingcap/TiProxy/lib/config"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

var serviceNotReady = regexp.MustCompile("service not ready")

// TiProxyControlInterface is the interface that knows how to control tiproxy clusters
type TiProxyControlInterface interface {
	// IsPodHealthy gets the healthy status of TiProxy pod.
	IsPodHealthy(tc *v1alpha1.TidbCluster, ordinal int32) (ok bool, err error)
	// SetConfigProxy set the proxy part in config
	SetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32, cfg *config.ProxyServerOnline) error
	// GetConfigProxy get the proxy part in config
	GetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32) (*config.ProxyServerOnline, error)
}

// defaultTiProxyControl is default implementation of TiProxyControlInterface.
type defaultTiProxyControl struct {
	secretLister corelisterv1.SecretLister
}

// NewDefaultTiProxyControl returns a defaultTiProxyControl instance
func NewDefaultTiProxyControl(secretLister corelisterv1.SecretLister) *defaultTiProxyControl {
	return &defaultTiProxyControl{secretLister: secretLister}
}

func (c *defaultTiProxyControl) getCli(tc *v1alpha1.TidbCluster, ordinal int32) func(io.Reader, ...string) (*bytes.Buffer, error) {
	return func(in io.Reader, s ...string) (*bytes.Buffer, error) {
		name := tc.GetName()
		ns := tc.GetNamespace()
		if tc.Heterogeneous() && tc.Spec.TiProxy == nil {
			name = tc.Spec.Cluster.Name
			ns = tc.Spec.Cluster.Namespace
		}

		args := append([]string{},
			"--log_level",
			"error",
			"--curls",
			fmt.Sprintf("%s.%s:3080", TiProxyPeerMemberName(name), ns),
		)
		var cmd *cobra.Command

		if !tc.IsTLSClusterEnabled() {
			cmd = cli.GetRootCmd(nil)
		} else {

			ns := tc.Namespace
			secretName := util.ClusterClientTLSSecretName(name)
			secret, err := c.secretLister.Secrets(ns).Get(secretName)
			if err != nil {
				return nil, err
			}

			clientCert, certExists := secret.Data[v1.TLSCertKey]
			clientKey, keyExists := secret.Data[v1.TLSPrivateKeyKey]
			if !certExists || !keyExists {
				return nil, fmt.Errorf("cert or key does not exist in secret %s/%s", ns, secretName)
			}

			tlsCert, err := tls.X509KeyPair(clientCert, clientKey)
			if err != nil {
				return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", ns, secretName, err)
			}

			rootCAs := x509.NewCertPool()
			rootCAs.AppendCertsFromPEM(secret.Data[v1.ServiceAccountRootCAKey])

			cmd = cli.GetRootCmd(&tls.Config{
				RootCAs:      rootCAs,
				Certificates: []tls.Certificate{tlsCert},
			})
		}

		out := new(bytes.Buffer)
		cmd.SetIn(in)
		cmd.SetOut(out)
		cmd.SetErr(out)
		cmd.SetArgs(append(args, s...))
		return out, cmd.Execute()
	}
}

func (c *defaultTiProxyControl) IsPodHealthy(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	out, err := c.getCli(tc, ordinal)(nil, "config", "proxy", "get")
	if err != nil {
		return false, err
	}
	return !serviceNotReady.MatchString(out.String()), nil
}

func (c *defaultTiProxyControl) GetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32) (*config.ProxyServerOnline, error) {
	out, err := c.getCli(tc, ordinal)(nil, "config", "proxy", "get")
	if err != nil {
		return nil, err
	}

	ret := &config.ProxyServerOnline{}
	if err := json.Unmarshal(out.Bytes(), ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *defaultTiProxyControl) SetConfigProxy(tc *v1alpha1.TidbCluster, ordinal int32, cfg *config.ProxyServerOnline) error {
	be, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	_, err = c.getCli(tc, ordinal)(bytes.NewReader(be), "config", "proxy", "set")
	return err
}
