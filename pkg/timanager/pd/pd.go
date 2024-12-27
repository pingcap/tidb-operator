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

package pd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/pkg/timanager/apis/pd/v1"
	tlsutil "github.com/pingcap/tidb-operator/pkg/utils/tls"
)

const (
	pdRequestTimeout = 10 * time.Second
)

type PDClientManager = timanager.Manager[*v1alpha1.PDGroup, PDClient]

type PDClient interface {
	HasSynced() bool
	Stores() StoreCache
	Members() MemberCache
	// TODO: only returns write interface
	Underlay() pdapi.PDClient
}

type pdClient struct {
	underlay pdapi.PDClient

	stores  StoreCache
	members MemberCache

	hasSynced []func() bool
}

func (c *pdClient) Stores() StoreCache {
	return c.stores
}

func (c *pdClient) Members() MemberCache {
	return c.members
}

func (c *pdClient) Underlay() pdapi.PDClient {
	return c.underlay
}

func (c *pdClient) HasSynced() bool {
	for _, f := range c.hasSynced {
		if !f() {
			return false
		}
	}

	return true
}

func NewClient(key string, underlay pdapi.PDClient, informerFactory timanager.SharedInformerFactory[pdapi.PDClient]) PDClient {
	storeInformer := informerFactory.InformerFor(&pdv1.Store{})
	memberInformer := informerFactory.InformerFor(&pdv1.Member{})
	stores := NewStoreCache(key, informerFactory)
	members := NewMemberCache(key, informerFactory)

	return &pdClient{
		underlay: underlay,
		stores:   stores,
		members:  members,
		hasSynced: []func() bool{
			storeInformer.HasSynced,
			memberInformer.HasSynced,
		},
	}
}

func PrimaryKey(ns, cluster string) string {
	return ns + ":" + cluster
}

func SplitPrimaryKey(key string) (ns, cluster string) {
	keys := strings.SplitN(key, ":", 2)
	if len(keys) < 2 {
		return keys[0], ""
	}
	return keys[0], keys[1]
}

// If any keys are changed, client will be renewed
// The first key is primary key to get client from manager
func CacheKeys(pdg *v1alpha1.PDGroup) ([]string, error) {
	keys := []string{}

	keys = append(keys,
		PrimaryKey(pdg.Namespace, pdg.Spec.Cluster.Name), // cluster name as primary key
		pdg.Name,
		string(pdg.GetUID()))
	// TODO: support reload tls config

	return keys, nil
}

var NewUnderlayClientFunc = func(c client.Client) timanager.NewUnderlayClientFunc[*v1alpha1.PDGroup, pdapi.PDClient] {
	return func(pdg *v1alpha1.PDGroup) (pdapi.PDClient, error) {
		ctx := context.Background()
		var cluster v1alpha1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      pdg.Spec.Cluster.Name,
			Namespace: pdg.Namespace,
		}, &cluster); err != nil {
			return nil, fmt.Errorf("cannot find cluster %s: %w", pdg.Spec.Cluster.Name, err)
		}

		host := fmt.Sprintf("%s-pd.%s:%d", pdg.Name, pdg.Namespace, pdg.GetClientPort())

		if cluster.IsTLSClusterEnabled() {
			tlsConfig, err := tlsutil.GetTLSConfigFromSecret(ctx, c,
				cluster.Namespace, v1alpha1.TLSClusterClientSecretName(cluster.Name))
			if err != nil {
				return nil, fmt.Errorf("cannot get tls config from secret: %w", err)
			}

			addr := "https://" + host
			return pdapi.NewPDClient(addr, pdRequestTimeout, tlsConfig), nil
		}

		addr := "http://" + host
		pc := pdapi.NewPDClient(addr, pdRequestTimeout, nil)
		return pc, nil
	}
}

func NewPDClientManager(_ logr.Logger, c client.Client) PDClientManager {
	m := timanager.NewManagerBuilder[*v1alpha1.PDGroup, pdapi.PDClient, PDClient]().
		WithNewUnderlayClientFunc(NewUnderlayClientFunc(c)).
		WithNewClientFunc(NewClient).
		WithCacheKeysFunc(CacheKeys).
		WithNewPollerFunc(&pdv1.Store{}, NewStorePoller).
		WithNewPollerFunc(&pdv1.Member{}, NewMemberPoller).
		Build()

	return m
}
