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

package tiflash

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/tiflashapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
)

type TiFlashClientManager = timanager.ClientManager[*v1alpha1.TiFlash, tiflashapi.TiFlashClient]

const (
	defaultTimeout = 10 * time.Second
)

func NewClient(
	f *v1alpha1.TiFlash,
	underlay tiflashapi.TiFlashClient,
	_ timanager.SharedInformerFactory[tiflashapi.TiFlashClient],
) tiflashapi.TiFlashClient {
	return underlay
}

func Key(f *v1alpha1.TiFlash) string {
	return timanager.PrimaryKey(f.Namespace, f.Name)
}

// CacheKeys returns the keys of the TiFlash.
// If any keys are changed, client will be renewed
// The first key is primary key to get client from manager
func CacheKeys(f *v1alpha1.TiFlash) ([]string, error) {
	var keys []string

	keys = append(keys,
		// instance name as primary key
		Key(f),
		string(f.GetUID()),
	)
	// TODO: support reload tls config

	return keys, nil
}

var NewUnderlayClientFunc = func(c client.Client) timanager.NewUnderlayClientFunc[*v1alpha1.TiFlash, tiflashapi.TiFlashClient] {
	return func(f *v1alpha1.TiFlash) (tiflashapi.TiFlashClient, error) {
		ctx := context.Background()
		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      f.Spec.Cluster.Name,
			Namespace: f.Namespace,
		}, &cluster); err != nil {
			return nil, fmt.Errorf("cannot find cluster %s: %w", f.Spec.Cluster.Name, err)
		}

		if coreutil.IsTLSClusterEnabled(&cluster) {
			tlsConfig, err := apicall.GetClientTLSConfig[scope.TiFlash](ctx, c, f)
			if err != nil {
				return nil, fmt.Errorf("cannot get tls config from secret: %w", err)
			}

			url := coreutil.TiFlashProxyStatusURL(f, true)
			return tiflashapi.NewTiFlashClient(url, defaultTimeout, tlsConfig), nil
		}

		url := coreutil.TiFlashProxyStatusURL(f, false)
		return tiflashapi.NewTiFlashClient(url, defaultTimeout, nil), nil
	}
}

func NewTiFlashClientManager(logger logr.Logger, c client.Client) TiFlashClientManager {
	m := timanager.NewManagerBuilder[*v1alpha1.TiFlash, tiflashapi.TiFlashClient, tiflashapi.TiFlashClient]().
		WithLogger(logger).
		WithNewUnderlayClientFunc(NewUnderlayClientFunc(c)).
		WithNewClientFunc(NewClient).
		WithCacheKeysFunc(CacheKeys).
		Build()

	return m
}
