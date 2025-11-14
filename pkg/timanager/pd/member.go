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
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
)

const (
	defaultPollInterval = 30 * time.Second
)

type (
	MemberCache = timanager.RefreshableCacheLister[pdv1.Member, *pdv1.Member]
)

func NewMemberCache(cluster string, informerFactory timanager.SharedInformerFactory[pdapi.PDClient]) MemberCache {
	informer := informerFactory.InformerFor(&pdv1.Member{})
	lister := timanager.NewGlobalCacheLister[pdv1.Member](
		informer.GetIndexer(),
		schema.GroupResource{
			Group:    pdv1.GroupName,
			Resource: "members",
		},
	).ByCluster(cluster)
	return timanager.CacheWithRefresher(lister, timanager.RefreshFunc(func() {
		informerFactory.Refresh(&pdv1.Member{})
	}))
}

func NewMemberPoller(name string, logger logr.Logger, c pdapi.PDClient) timanager.Poller {
	lister := NewMemberLister(name, c)

	// TODO: change interval
	return timanager.NewPoller(name, logger, lister, timanager.NewDeepEquality[pdv1.Member](), defaultPollInterval)
}

type memberLister struct {
	cluster string
	c       pdapi.PDClient
}

func NewMemberLister(cluster string, c pdapi.PDClient) timanager.Lister[pdv1.Member, *pdv1.Member, *pdv1.MemberList] {
	return &memberLister{
		cluster: cluster,
		c:       c,
	}
}

func (l *memberLister) List(ctx context.Context) (*pdv1.MemberList, error) {
	info, err := l.c.GetMembers(ctx)
	if err != nil {
		return nil, err
	}

	health, err := l.c.GetHealth(ctx)
	if err != nil {
		return nil, err
	}

	mm := map[uint64]*pdv1.Member{}

	for _, m := range info.Members {
		mm[m.MemberId] = &pdv1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.Name,
				Namespace: l.cluster,
			},
			ClusterID:      strconv.FormatUint(info.Header.ClusterId, 10),
			ID:             strconv.FormatUint(m.MemberId, 10),
			PeerUrls:       m.PeerUrls,
			ClientUrls:     m.ClientUrls,
			LeaderPriority: m.LeaderPriority,

			IsLeader:     m.MemberId == info.Leader.MemberId,
			IsEtcdLeader: m.MemberId == info.EtcdLeader.MemberId,
		}
	}

	for _, h := range health.Healths {
		m, ok := mm[h.MemberID]
		if !ok {
			return nil, fmt.Errorf("member %s(%v) doesn't exist but return health info", h.Name, h.MemberID)
		}
		m.Health = h.Health
		mm[h.MemberID] = m
	}

	list := pdv1.MemberList{}

	for _, m := range mm {
		list.Items = append(list.Items, *m)
	}

	slices.SortFunc(list.Items, func(a, b pdv1.Member) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &list, nil
}

func (*memberLister) GetItems(list *pdv1.MemberList) []*pdv1.Member {
	objs := make([]*pdv1.Member, 0, len(list.Items))
	for i := range list.Items {
		objs = append(objs, &list.Items[i])
	}

	return objs
}

func (*memberLister) MarkAsInvalid(m *pdv1.Member) bool {
	if !m.Invalid {
		m.Invalid = true
		return true
	}
	return false
}
