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

package resourcemanager

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/resourcemanagerapi"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
)

const (
	resourceManagerRequestTimeout = 10 * time.Second
)

type ResourceManagerClientManager = timanager.Manager[*v1alpha1.ResourceManagerGroup, ResourceManagerClient]

type ResourceManagerClient interface {
	Underlay() resourcemanagerapi.Client
}

type resourceManagerClient struct {
	underlay resourcemanagerapi.Client
}

func (c *resourceManagerClient) Underlay() resourcemanagerapi.Client {
	return c.underlay
}

func NewClient(
	_ *v1alpha1.ResourceManagerGroup,
	underlay resourcemanagerapi.Client,
	_ timanager.SharedInformerFactory[resourcemanagerapi.Client],
) (ResourceManagerClient, error) {
	return &resourceManagerClient{underlay: underlay}, nil
}

// CacheKeys returns the keys of the ResourceManagerGroup.
// If any keys are changed, client will be renewed.
// The first key is primary key to get client from manager.
func CacheKeys(rmg *v1alpha1.ResourceManagerGroup) ([]string, error) {
	keys := []string{
		// cluster name as primary key
		timanager.PrimaryKey(rmg.Namespace, rmg.Spec.Cluster.Name),
		rmg.Name,
		string(rmg.GetUID()),
	}

	return keys, nil
}

func NewUnderlayClientFunc(c client.Client) timanager.NewUnderlayClientFunc[*v1alpha1.ResourceManagerGroup, resourcemanagerapi.Client] {
	return func(rmg *v1alpha1.ResourceManagerGroup) (resourcemanagerapi.Client, error) {
		ctx, cancel := context.WithTimeout(context.Background(), resourceManagerRequestTimeout)
		defer cancel()

		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      rmg.Spec.Cluster.Name,
			Namespace: rmg.Namespace,
		}, &cluster); err != nil {
			return nil, fmt.Errorf("cannot find cluster %s: %w", rmg.Spec.Cluster.Name, err)
		}

		scheme := "http"
		var tlsConfig *tls.Config
		if coreutil.IsTLSClusterEnabled(&cluster) {
			cfg, err := apicall.GetClientTLSConfig(ctx, c, &cluster)
			if err != nil {
				return nil, fmt.Errorf("cannot get tls config from secret: %w", err)
			}
			tlsConfig = cfg
			scheme = "https"
		}

		svcName := coreutil.HeadlessServiceName[scope.ResourceManagerGroup](rmg)
		url := fmt.Sprintf("%s://%s.%s.svc:%d", scheme, svcName, rmg.Namespace, coreutil.ResourceManagerGroupClientPort(rmg))
		return resourcemanagerapi.NewClient(url, resourceManagerRequestTimeout, tlsConfig), nil
	}
}

func NewResourceManagerClientManager(logger logr.Logger, c client.Client) ResourceManagerClientManager {
	m := timanager.NewManagerBuilder[*v1alpha1.ResourceManagerGroup, resourcemanagerapi.Client, ResourceManagerClient]().
		WithLogger(logger).
		WithNewUnderlayClientFunc(NewUnderlayClientFunc(c)).
		WithNewClientFunc(NewClient).
		WithCacheKeysFunc(CacheKeys).
		Build()

	return m
}
