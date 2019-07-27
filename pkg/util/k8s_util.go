// Copyright 2019 PingCAP, Inc.
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

package util

import (
	"os"

	"github.com/golang/glog"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const KUBECONFIGENV = "KUBECONFIG"

func NewClusterConfig(masterURL, kubeconfig string) (cfg *rest.Config) {
	var err error
	if os.Getenv(KUBECONFIGENV) != "" {
		kubeconfig = os.Getenv(KUBECONFIGENV)
	}
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			glog.Fatalf("failed to create config from file %s, masterURL %s: %v\n", kubeconfig, masterURL, err)
		}
		return
	}
	cfg, err = rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("failed to get in-cluster config: %v", err)
	}
	return
}
