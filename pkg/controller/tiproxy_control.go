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
	"fmt"
	"io"

	"github.com/pingcap/TiProxy/lib/cli"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/spf13/cobra"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

// TiProxyControlInterface is the interface that knows how to control tiproxy clusters
type TiProxyControlInterface interface {
	// IsHealth check if node is healthy.
	IsHealth(tc *v1alpha1.TidbCluster, ordinal int32) (*bytes.Buffer, error)
}

var _ TiProxyControlInterface = &defaultTiProxyControl{}

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
			// using certs of TiDB with tiproxy name, we don't have a correct CA
			cmd = cli.GetRootCmd(&tls.Config{
				InsecureSkipVerify: true,
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

func (c *defaultTiProxyControl) IsHealth(tc *v1alpha1.TidbCluster, ordinal int32) (*bytes.Buffer, error) {
	return c.getCli(tc, ordinal)(nil, "health")
}
