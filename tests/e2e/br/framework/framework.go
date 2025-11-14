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

package framework

import (
	"context"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/br/utils/s3"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/config"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/k8s"
)

type Framework struct {
	*framework.Framework

	// PortForwarder is defined to visit pod in local.
	PortForwarder k8s.PortForwarder

	// Storage defines interface of s3 storage
	Storage s3.Interface

	YamlApplier *k8s.YAMLApplier
}

func LoadClientRawConfig() (clientcmdapi.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).RawConfig()
}

func NewFramework(baseName string) *Framework {
	f := &Framework{
		Framework: framework.New(),
	}
	// skip waiting for namespace and cluster to be deleted, because we don't care about it in BR e2e tests
	f.Framework.Setup(
		framework.WithSkipWaitForNamespaceDeleted(),
		framework.WithSkipWaitForClusterDeleted(),
	)

	var err error
	// TODO: only run once
	clientRawConfig, err := LoadClientRawConfig()
	gomega.Expect(err).To(gomega.Succeed())
	f.PortForwarder, err = k8s.NewPortForwarder(context.Background(), config.NewSimpleRESTClientGetter(&clientRawConfig))
	gomega.Expect(err).To(gomega.Succeed())

	provider := "kind"
	f.Storage, err = s3.New(provider, f.Client, f.PortForwarder)
	gomega.Expect(err).To(gomega.Succeed())

	conf, err := framework.NewConfig("", "")
	gomega.Expect(err).To(gomega.Succeed())
	// build yaml applier
	clientSet, _, restConfig := initK8sClient()
	_ = restConfig
	dynamicClient := dynamic.NewForConfigOrDie(conf)
	cachedDiscovery := cacheddiscovery.NewMemCacheClient(clientSet.Discovery())
	restmapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)
	restmapper.Reset()
	f.YamlApplier = k8s.NewYAMLApplier(dynamicClient, restmapper)

	return f
}

func initK8sClient() (kubernetes.Interface, client.Client, *rest.Config) {
	restConfig, err := k8s.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	clientset, err := kubernetes.NewForConfig(restConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	scheme := runtime.NewScheme()
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	gomega.Expect(clientgoscheme.AddToScheme(scheme)).To(gomega.Succeed())
	gomega.Expect(v1alpha1.Install(scheme)).To(gomega.Succeed())

	// also init a controller-runtime client
	k8sClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return clientset, k8sClient, restConfig
}
