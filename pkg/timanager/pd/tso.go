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
	"cmp"
	"context"
	"slices"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
)

type (
	TSOMemberCache = timanager.RefreshableCacheLister[pdv1.TSOMember, *pdv1.TSOMember]
)

func NewTSOMemberCache(cluster string, informerFactory timanager.SharedInformerFactory[pdapi.PDClient]) TSOMemberCache {
	informer := informerFactory.InformerFor(&pdv1.TSOMember{})
	lister := timanager.NewGlobalCacheLister[pdv1.TSOMember](
		informer.GetIndexer(),
		schema.GroupResource{
			Group:    pdv1.GroupName,
			Resource: "tsomembers",
		},
	).ByCluster(cluster)
	return timanager.CacheWithRefresher(lister, timanager.RefreshFunc(func() {
		informerFactory.Refresh(&pdv1.TSOMember{})
	}))
}

func NewTSOMemberPoller(name string, logger logr.Logger, c pdapi.PDClient) timanager.Poller {
	lister := NewTSOMemberLister(name, c)

	// TODO: change interval
	return timanager.NewPoller(name, logger, lister, timanager.NewDeepEquality[pdv1.TSOMember](), defaultPollInterval)
}

type tsoMemberLister struct {
	cluster string
	c       pdapi.PDClient
}

func NewTSOMemberLister(cluster string, c pdapi.PDClient) timanager.Lister[pdv1.TSOMember, *pdv1.TSOMember, *pdv1.TSOMemberList] {
	return &tsoMemberLister{
		cluster: cluster,
		c:       c,
	}
}

func (l *tsoMemberLister) List(ctx context.Context) (*pdv1.TSOMemberList, error) {
	info, err := l.c.GetTSOMembers(ctx)
	if err != nil {
		return nil, err
	}
	primary, err := l.c.GetTSOLeader(ctx)
	if err != nil {
		return nil, err
	}

	list := pdv1.TSOMemberList{}

	for _, m := range info {
		list.Items = append(list.Items, pdv1.TSOMember{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.Name,
				Namespace: l.cluster,
			},
			ServiceAddr:    m.ServiceAddr,
			Version:        m.Version,
			StartTimestamp: m.StartTimestamp,

			IsLeader: m.ServiceAddr == primary,
		})
	}

	slices.SortFunc(list.Items, func(a, b pdv1.TSOMember) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &list, nil
}

func (*tsoMemberLister) GetItems(list *pdv1.TSOMemberList) []*pdv1.TSOMember {
	objs := make([]*pdv1.TSOMember, 0, len(list.Items))
	for i := range list.Items {
		objs = append(objs, &list.Items[i])
	}

	return objs
}

func (*tsoMemberLister) MarkAsInvalid(m *pdv1.TSOMember) bool {
	if !m.Invalid {
		m.Invalid = true
		return true
	}
	return false
}
