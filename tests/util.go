// Copyright 2018 PingCAP, Inc.
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
// limitations under the License.package spec

package tests

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CreateKubeClient() (versioned.Interface, kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, err
	}
	cfg.Timeout = 30 * time.Second
	operatorCli, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return operatorCli, kubeCli, nil
}

// func ()
