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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination mock_generated.go -package=pd ${GO_MODULE}/pkg/timanager/pd PDClient,StoreCache,MemberCache
package pd

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
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
)

const (
	defaultTimeout = 10 * time.Second
)

type PDClientManager = timanager.Manager[*v1alpha1.Cluster, PDClient]

type PDClient interface {
	HasSynced() bool
	Stores() StoreCache
	Members() MemberCache
	TSOMembers() TSOMemberCache
	// TODO: only returns write interface
	Underlay() pdapi.PDClient
}

type pdClient struct {
	underlay pdapi.PDClient

	stores     StoreCache
	members    MemberCache
	tsoMembers TSOMemberCache

	hasSynced []func() bool
}

func (c *pdClient) Stores() StoreCache {
	return c.stores
}

func (c *pdClient) Members() MemberCache {
	return c.members
}

func (c *pdClient) TSOMembers() TSOMemberCache {
	return c.tsoMembers
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

func NewClientFunc(c client.Client) timanager.NewClientFunc[*v1alpha1.Cluster, pdapi.PDClient, PDClient] {
	return func(
		cluster *v1alpha1.Cluster,
		underlay pdapi.PDClient,
		informerFactory timanager.SharedInformerFactory[pdapi.PDClient],
	) (PDClient, error) {
		mode, err := getPDMode(c, cluster)
		if err != nil {
			return nil, err
		}
		storeInformer := informerFactory.InformerFor(&pdv1.Store{})
		memberInformer := informerFactory.InformerFor(&pdv1.Member{})

		key := timanager.PrimaryKey(cluster.Namespace, cluster.Name)

		stores := NewStoreCache(key, informerFactory)
		members := NewMemberCache(key, informerFactory)

		pdc := &pdClient{
			underlay: underlay,
			stores:   stores,
			members:  members,
			hasSynced: []func() bool{
				storeInformer.HasSynced,
				memberInformer.HasSynced,
			},
		}

		if mode == v1alpha1.PDModeMS {
			tsoMemberInformer := informerFactory.InformerFor(&pdv1.TSOMember{})
			tsoMembers := NewTSOMemberCache(key, informerFactory)
			pdc.tsoMembers = tsoMembers
			pdc.hasSynced = append(pdc.hasSynced, tsoMemberInformer.HasSynced)
		}

		return pdc, nil
	}
}

// CacheKeysFunc returns the keys func of the Cluster.
// If any keys are changed, client will be renewed
// The first key is primary key to get client from manager
func CacheKeysFunc(c client.Client) func(cluster *v1alpha1.Cluster) ([]string, error) {
	return func(cluster *v1alpha1.Cluster) ([]string, error) {
		mode, err := getPDMode(c, cluster)
		if err != nil {
			return nil, err
		}

		var keys []string

		keys = append(keys,
			// cluster name as primary key
			timanager.PrimaryKey(cluster.Namespace, cluster.Name),
			// if mode is changed, renew client
			string(mode),
		)
		// TODO: support reload tls config

		return keys, nil
	}
}

// If there are any PD in ms mode, use ms mode client
func getPDMode(c client.Client, cluster *v1alpha1.Cluster) (v1alpha1.PDMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	mode := v1alpha1.PDModeNormal

	pdgs, err := apicall.ListGroups[scope.PDGroup](ctx, c, cluster.Namespace, cluster.Name)
	if err != nil {
		return mode, fmt.Errorf("cannot list pd groups: %w", err)
	}

	for _, pdg := range pdgs {
		if pdg.Spec.Template.Spec.Mode == v1alpha1.PDModeMS {
			mode = v1alpha1.PDModeMS
		}
	}

	return mode, nil
}

var NewUnderlayClientFunc = func(c client.Client) timanager.NewUnderlayClientFunc[*v1alpha1.Cluster, pdapi.PDClient] {
	return func(cluster *v1alpha1.Cluster) (pdapi.PDClient, error) {
		if cluster.Status.PD == "" {
			return nil, fmt.Errorf("no pd addr, waiting until pd groups are created")
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		var tlsConfig *tls.Config
		if coreutil.IsTLSClusterEnabled(cluster) {
			cfg, err := apicall.GetClientTLSConfig(ctx, c, cluster)
			if err != nil {
				return nil, fmt.Errorf("cannot get tls config from secret: %w", err)
			}
			tlsConfig = cfg
		}

		pc := pdapi.NewPDClient(cluster.Status.PD, defaultTimeout, tlsConfig)
		return pc, nil
	}
}

func NewPDClientManager(logger logr.Logger, c client.Client) PDClientManager {
	m := timanager.NewManagerBuilder[*v1alpha1.Cluster, pdapi.PDClient, PDClient]().
		WithLogger(logger).
		WithNewUnderlayClientFunc(NewUnderlayClientFunc(c)).
		WithNewClientFunc(NewClientFunc(c)).
		WithCacheKeysFunc(CacheKeysFunc(c)).
		WithNewPollerFunc(&pdv1.Store{}, NewStorePoller).
		WithNewPollerFunc(&pdv1.Member{}, NewMemberPoller).
		WithNewPollerFunc(&pdv1.TSOMember{}, NewTSOMemberPoller).
		Build()

	return m
}
