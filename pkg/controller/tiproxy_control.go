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
	"errors"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tiproxy/lib/cli"
	"github.com/spf13/cobra"
)

// TiProxyControlInterface is the interface that knows how to control tiproxy clusters
type TiProxyControlInterface interface {
	// IsHealth check if node is healthy.
	IsHealth(tc *v1alpha1.TidbCluster, ordinal int32) (*bytes.Buffer, error)
	// SetLabels sets labels for a tiproxy pod
	SetLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error
}

var _ TiProxyControlInterface = &defaultTiProxyControl{}

// defaultTiProxyControl is default implementation of TiProxyControlInterface.
type defaultTiProxyControl struct {
	testURL string
}

// NewDefaultTiProxyControl returns a defaultTiProxyControl instance
func NewDefaultTiProxyControl() *defaultTiProxyControl {
	return &defaultTiProxyControl{}
}

func (c *defaultTiProxyControl) getCli(tc *v1alpha1.TidbCluster, ordinal int32) func(io.Reader, ...string) (*bytes.Buffer, error) {
	return func(in io.Reader, s ...string) (*bytes.Buffer, error) {
		args := append([]string{},
			"--log_level",
			"error",
			"--curls",
			c.getBaseURL(tc, ordinal),
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

func (c *defaultTiProxyControl) getBaseURL(tc *v1alpha1.TidbCluster, ordinal int32) string {
	if len(c.testURL) > 0 {
		return c.testURL
	}

	tcName := tc.GetName()
	ns := tc.GetNamespace()
	if tc.Heterogeneous() && tc.Spec.TiProxy == nil {
		tcName = tc.Spec.Cluster.Name
		ns = tc.Spec.Cluster.Namespace
	}
	memberName := TiProxyMemberName(tcName)
	hostName := fmt.Sprintf("%s-%d", memberName, ordinal)
	peerName := TiProxyPeerMemberName(tcName)
	if tc.Spec.ClusterDomain != "" {
		return fmt.Sprintf("%s.%s.%s.svc.%s:%d", hostName, peerName, ns, tc.Spec.ClusterDomain, v1alpha1.DefaultTiProxyStatusPort)
	}
	return fmt.Sprintf("%s.%s.%s:%d", hostName, peerName, ns, v1alpha1.DefaultTiProxyStatusPort)
}

func (c *defaultTiProxyControl) IsHealth(tc *v1alpha1.TidbCluster, ordinal int32) (*bytes.Buffer, error) {
	return c.getCli(tc, ordinal)(nil, "health")
}

func (c *defaultTiProxyControl) SetLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error {
	type labelConfig struct {
		Labels map[string]string `toml:"labels"`
	}
	cfg := labelConfig{Labels: labels}
	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(cfg); err != nil {
		return fmt.Errorf("encode labels to toml failed, error: %v", err)
	}

	_, err := c.getCli(tc, ordinal)(&buffer, "config", "set")
	return err
}

type FakeTiProxyControl struct {
	healthInfo     map[string]string
	setLabelsError error
}

// NewFakeTiProxyControl returns a FakeTiProxyControl instance
func NewFakeTiProxyControl() *FakeTiProxyControl {
	return &FakeTiProxyControl{}
}

// SetHealth sets health info for FakeTiProxyControl
func (c *FakeTiProxyControl) SetHealth(healthInfo map[string]string) {
	c.healthInfo = healthInfo
}

func (c *FakeTiProxyControl) SetLabelsErr(err error) {
	c.setLabelsError = err
}

func (c *FakeTiProxyControl) IsHealth(tc *v1alpha1.TidbCluster, ordinal int32) (*bytes.Buffer, error) {
	podName := fmt.Sprintf("%s-%d", TiProxyMemberName(tc.GetName()), ordinal)
	if c.healthInfo == nil {
		return nil, errors.New("health info not set")
	}
	if health, ok := c.healthInfo[podName]; ok {
		return bytes.NewBuffer([]byte(health)), nil
	}
	return nil, errors.New("health info not set")
}

func (c *FakeTiProxyControl) SetLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error {
	return c.setLabelsError
}
