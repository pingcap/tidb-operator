// Copyright 2023 PingCAP, Inc.
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

package kube

import (
	"github.com/pingcap/log"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
)

const (
	defaultContext = "default"
)

type KubeClient struct {
	// inCluster indicates whether this app is running in k8s cluster.
	// When it's true, no `kubenetes-id` for some APIs is needed.
	inCluster bool

	clients map[string]versioned.Interface // context --> pingcap client
}

func (kc *KubeClient) GetClient(context string) versioned.Interface {
	if context == "" && kc.inCluster {
		context = defaultContext
	}
	return kc.clients[context]
}

func InitKubeClients(kubeconfigPath string) (*KubeClient, error) {
	kc := &KubeClient{
		clients: make(map[string]versioned.Interface),
	}
	ctxNames := make([]string, 0)

	if kubeconfigPath != "" {
		kubeConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return nil, err
		}

		for contextName := range kubeConfig.Contexts {
			cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
				&clientcmd.ConfigOverrides{CurrentContext: contextName}).ClientConfig()
			if err != nil {
				return nil, err // return error if any kube client init failed
			}

			// we use the same QPS and Burst as for the API server which is running this manager now
			cfg.QPS = float32(100)
			cfg.Burst = 100

			cli, err := versioned.NewForConfig(cfg)
			if err != nil {
				return nil, err
			}
			kc.clients[contextName] = cli
			ctxNames = append(ctxNames, contextName)
		}
	} else {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		cfg.QPS = float32(100)
		cfg.Burst = 100

		cli, err := versioned.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}
		kc.clients[defaultContext] = cli
		kc.inCluster = true // in k8s cluster
		ctxNames = append(ctxNames, defaultContext)
	}

	log.Info("kube clients init success", zap.Bool("inCluster", kc.inCluster), zap.Strings("k8s-contexts", ctxNames))

	return kc, nil
}
