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
	"fmt"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/scheme"
)

func newClient(configPath, ctxName string) (client.Client, error) {
	rule := clientcmd.NewDefaultClientConfigLoadingRules()
	rule.ExplicitPath = configPath

	kubeconfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rule,
		&clientcmd.ConfigOverrides{CurrentContext: ctxName}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("can't parse kubeconfig: %w", err)
	}
	c, err := client.New(kubeconfig, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("can't new client: %w", err)
	}
	return c, nil
}
