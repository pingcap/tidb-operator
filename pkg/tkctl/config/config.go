// Copyright 2019. PingCAP, Inc.
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

package config

import (
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/user"
	"path/filepath"
	"sync"
)

const (
	tcConfigRelativePath = "/.kube/tidbcluster-config"
)

// TkcContext wraps the configuration and credential for tidb cluster accessing.
type TkcContext struct {
	*genericclioptions.ConfigFlags

	TidbClusterConfig *TidbClusterConfig

	loadingLock  sync.Mutex
	clientConfig clientcmd.ClientConfig
}

// NewTkcContext create a TkcContext
func NewTkcContext(kubeFlags *genericclioptions.ConfigFlags) *TkcContext {
	return &TkcContext{ConfigFlags: kubeFlags}
}

// TkcBuilder returns a builder that operates on generic objects under tkc context
func (c *TkcContext) TkcBuilder() *resource.Builder {
	return resource.NewBuilder(c)
}

// KubeBuilder returns a builder that operates on generic objects under original kubectl context
func (c *TkcContext) KubeBuilder() *resource.Builder {
	return resource.NewBuilder(c.ConfigFlags)
}

// ToTkcConfigLoader create the tkc client config for tidb cluster which overrides
// the tidb cluster context and namespace to the raw kubectl config
func (c *TkcContext) ToTkcConfigLoader() (clientcmd.ClientConfig, error) {
	if c.clientConfig != nil {
		return c.clientConfig, nil
	}

	// config has not loaded, do loading
	c.loadingLock.Lock()
	defer c.loadingLock.Unlock()
	// double check after lock acquired
	if c.clientConfig != nil {
		return c.clientConfig, nil
	}

	// try loading tidbcluster config
	tcConfigFile, err := tcConfigLocation()
	if err != nil {
		glog.V(4).Info("Error getting tidb cluster config file location")
	} else {
		tcConfig, err := LoadFile(tcConfigFile)
		if err != nil {
			glog.V(4).Info("Error reading tidb cluster config file")
		} else {
			c.TidbClusterConfig = tcConfig
		}
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	if c.KubeConfig != nil {
		loadingRules.ExplicitPath = *c.KubeConfig
	}
	mergedConfig, err := loadingRules.Load()
	if err != nil {
		return nil, err
	}

	overrides := c.collectOverrides()

	var clientConfig clientcmd.ClientConfig

	// we only have an interactive prompt when a password is allowed
	if c.Password == nil {
		clientConfig = clientcmd.NewNonInteractiveClientConfig(*mergedConfig, overrides.CurrentContext, overrides, loadingRules)
	} else {
		clientConfig = clientcmd.NewInteractiveClientConfig(*mergedConfig, overrides.CurrentContext, overrides, os.Stdin, loadingRules)
	}

	return clientConfig, nil
}

// ToRestConfig overrides ConfigFlags.ToRestConfig()
func (c *TkcContext) ToRESTConfig() (*rest.Config, error) {
	configLoader, err := c.ToTkcConfigLoader()
	if err != nil {
		return nil, err
	}
	return configLoader.ClientConfig()
}

// ToKubectlRestConfig returns the rest config under kubectl context
func (c *TkcContext) ToKubectlRestConfig() (*rest.Config, error) {
	return c.ConfigFlags.ToRESTConfig()
}

// SwitchTidbCluster store current tidb cluster configuration to local file,
// this action will not affect the configured context and namespace.
func (c *TkcContext) SwitchTidbCluster(context, namespace, clusterName string) error {
	tcConfigFile, err := tcConfigLocation()
	if err != nil {
		return err
	}
	dir := filepath.Dir(tcConfigFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	tcConfig := &TidbClusterConfig{
		KubeContext: context,
		Namespace:   namespace,
		ClusterName: clusterName,
	}
	content, err := yaml.Marshal(tcConfig)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(tcConfigFile, content, 0644); err != nil {
		return err
	}
	return nil
}

func (c *TkcContext) collectOverrides() *clientcmd.ConfigOverrides {

	// calculate flag and config overrides
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

	// bind tidb cluster config and flags, command line flags has higher priority
	if c.Context != nil {
		overrides.CurrentContext = *c.Context
	} else if c.TidbClusterConfig != nil && len(c.TidbClusterConfig.KubeContext) > 0 {
		overrides.CurrentContext = c.TidbClusterConfig.KubeContext
	}
	if c.Namespace != nil {
		overrides.Context.Namespace = *c.Namespace
	} else if c.TidbClusterConfig != nil && len(c.TidbClusterConfig.Namespace) > 0 {
		overrides.Context.Namespace = c.TidbClusterConfig.Namespace
	}

	// bind other flag values to overrides
	if c.CertFile != nil {
		overrides.AuthInfo.ClientCertificate = *c.CertFile
	}
	if c.KeyFile != nil {
		overrides.AuthInfo.ClientKey = *c.KeyFile
	}
	if c.BearerToken != nil {
		overrides.AuthInfo.Token = *c.BearerToken
	}
	if c.Impersonate != nil {
		overrides.AuthInfo.Impersonate = *c.Impersonate
	}
	if c.ImpersonateGroup != nil {
		overrides.AuthInfo.ImpersonateGroups = *c.ImpersonateGroup
	}
	if c.Username != nil {
		overrides.AuthInfo.Username = *c.Username
	}
	if c.Password != nil {
		overrides.AuthInfo.Password = *c.Password
	}
	if c.APIServer != nil {
		overrides.ClusterInfo.Server = *c.APIServer
	}
	if c.CAFile != nil {
		overrides.ClusterInfo.CertificateAuthority = *c.CAFile
	}
	if c.Insecure != nil {
		overrides.ClusterInfo.InsecureSkipTLSVerify = *c.Insecure
	}
	if c.ClusterName != nil {
		overrides.Context.Cluster = *c.ClusterName
	}
	if c.AuthInfoName != nil {
		overrides.Context.AuthInfo = *c.AuthInfoName
	}
	if c.Timeout != nil {
		overrides.Timeout = *c.Timeout
	}

	return overrides
}

func tcConfigLocation() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir + tcConfigRelativePath, nil
}
