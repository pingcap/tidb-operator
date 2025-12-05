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
	"strconv"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
)

type (
	StoreCache = timanager.RefreshableCacheLister[pdv1.Store, *pdv1.Store]
)

func NewStoreCache(cluster string, informerFactory timanager.SharedInformerFactory[pdapi.PDClient]) StoreCache {
	informer := informerFactory.InformerFor(&pdv1.Store{})
	lister := timanager.NewGlobalCacheLister[pdv1.Store](
		informer.GetIndexer(),
		schema.GroupResource{
			Group:    pdv1.GroupName,
			Resource: "stores",
		},
	).ByCluster(cluster)
	return timanager.CacheWithRefresher(lister, timanager.RefreshFunc(func() {
		informerFactory.Refresh(&pdv1.Store{})
	}))
}

func NewStorePoller(name string, logger logr.Logger, c pdapi.PDClient) timanager.Poller {
	lister := NewStoreLister(name, c)

	// TODO: change interval
	return timanager.NewPoller(name, logger, lister, timanager.NewDeepEquality[pdv1.Store](logger), defaultPollInterval)
}

type storeLister struct {
	cluster string
	c       pdapi.PDClient
}

func NewStoreLister(cluster string, c pdapi.PDClient) timanager.Lister[pdv1.Store, *pdv1.Store, *pdv1.StoreList] {
	return &storeLister{
		cluster: cluster,
		c:       c,
	}
}

func (l *storeLister) List(ctx context.Context) (*pdv1.StoreList, error) {
	ss, err := l.c.GetStores(ctx)
	if err != nil {
		if pdapi.IsTiKVNotBootstrappedError(err) {
			return &pdv1.StoreList{}, nil
		}
		return nil, err
	}

	list := pdv1.StoreList{}
	for _, s := range ss.Stores {
		obj := l.convert(l.cluster, s)
		list.Items = append(list.Items, *obj)
	}
	return &list, nil
}

func (*storeLister) GetItems(list *pdv1.StoreList) []*pdv1.Store {
	objs := make([]*pdv1.Store, 0, len(list.Items))
	for i := range list.Items {
		objs = append(objs, &list.Items[i])
	}

	return objs
}

func (*storeLister) MarkAsInvalid(m *pdv1.Store) bool {
	if !m.Invalid {
		m.Invalid = true
		return true
	}
	return false
}

func (*storeLister) convert(cluster string, s *pdapi.StoreInfo) *pdv1.Store {
	ls := map[string]string{}
	for _, label := range s.Store.Labels {
		if label != nil {
			ls[label.Key] = label.Value
		}
	}

	return &pdv1.Store{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          ls,
			Name:            s.Store.Address,
			Namespace:       cluster,
			ResourceVersion: uuid.NewString(),
		},
		ID: strconv.FormatUint(s.Store.Id, 10),
		// Address:             s.Store.Address,
		Version:             s.Store.Version,
		PhysicallyDestroyed: s.Store.PhysicallyDestroyed,
		State:               pdv1.StoreState(s.Store.GetState().String()),
		NodeState:           pdv1.NodeState(s.Store.NodeState.String()),
		StartTimestamp:      s.Store.StartTimestamp,
		// LastHeartbeat:       s.Store.LastHeartbeat,
		IsBusy: s.Status.IsBusy,

		LeaderCount: s.Status.LeaderCount,
		RegionCount: s.Status.RegionCount,
	}
}
