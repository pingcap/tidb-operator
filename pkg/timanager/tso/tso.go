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

package tso

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	"github.com/pingcap/tidb-operator/v2/pkg/tsoapi"
)

const (
	tsoRequestTimeout = 10 * time.Second
)

type TSOClientManager = timanager.Manager[*v1alpha1.TSOGroup, TSOClient]

type TSOClient interface {
	Underlay() tsoapi.TSOClient
}

type tsoClient struct {
	underlay tsoapi.TSOClient
}

func (c *tsoClient) Underlay() tsoapi.TSOClient {
	return c.underlay
}

func NewClient(tg *v1alpha1.TSOGroup, underlay tsoapi.TSOClient, _ timanager.SharedInformerFactory[tsoapi.TSOClient]) (TSOClient, error) {
	c := &tsoClient{
		underlay: underlay,
	}
	return c, nil
}

// CacheKeys returns the keys of the TSOGroup.
// If any keys are changed, client will be renewed
// The first key is primary key to get client from manager
func CacheKeys(tg *v1alpha1.TSOGroup) ([]string, error) {
	var keys []string

	keys = append(keys,
		// cluster name as primary key
		timanager.PrimaryKey(tg.Namespace, tg.Spec.Cluster.Name),
		tg.Name,
		string(tg.GetUID()),
	)
	// TODO: support reload tls config

	return keys, nil
}

func NewUnderlayClientFunc(c client.Client) timanager.NewUnderlayClientFunc[*v1alpha1.TSOGroup, tsoapi.TSOClient] {
	return func(tg *v1alpha1.TSOGroup) (tsoapi.TSOClient, error) {
		ctx := context.Background()
		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      tg.Spec.Cluster.Name,
			Namespace: tg.Namespace,
		}, &cluster); err != nil {
			return nil, fmt.Errorf("cannot find cluster %s: %w", tg.Spec.Cluster.Name, err)
		}

		if coreutil.IsTLSClusterEnabled(&cluster) {
			tlsConfig, err := apicall.GetClientTLSConfig(ctx, c, &cluster)
			if err != nil {
				return nil, fmt.Errorf("cannot get tls config from secret: %w", err)
			}

			url := coreutil.TSOServiceURL(tg, true)
			return tsoapi.NewTSOClient(url, tsoRequestTimeout, tlsConfig), nil
		}

		url := coreutil.TSOServiceURL(tg, false)
		return tsoapi.NewTSOClient(url, tsoRequestTimeout, nil), nil
	}
}

func NewTSOClientManager(logger logr.Logger, c client.Client) TSOClientManager {
	m := timanager.NewManagerBuilder[*v1alpha1.TSOGroup, tsoapi.TSOClient, TSOClient]().
		WithLogger(logger).
		WithNewUnderlayClientFunc(NewUnderlayClientFunc(c)).
		WithNewClientFunc(NewClient).
		WithCacheKeysFunc(CacheKeys).
		Build()

	return m
}
