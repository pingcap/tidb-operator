// Copyright 2018 PingCAP, Inc.
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

package member

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
)

type tikvFailover struct {
	tikvFailoverPeriod time.Duration
}

// NewTiKVFailover returns a tikv Failover
func NewTiKVFailover(tikvFailoverPeriod time.Duration) Failover {
	return &tikvFailover{tikvFailoverPeriod}
}

func (tf *tikvFailover) Failover(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for storeID, store := range tc.Status.TiKV.Stores {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		deadline := store.LastTransitionTime.Add(tf.tikvFailoverPeriod)
		exist := false
		for _, failureStore := range tc.Status.TiKV.FailureStores {
			if failureStore.PodName == podName {
				exist = true
				break
			}
		}
		if store.State == v1alpha1.TiKVStateDown && time.Now().After(deadline) && !exist {
			if tc.Status.TiKV.FailureStores == nil {
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
			}
			if tc.Spec.TiKV.MaxFailoverCount != nil {
				maxFailoverCount := *tc.Spec.TiKV.MaxFailoverCount
				if maxFailoverCount > 0 && len(tc.Status.TiKV.FailureStores) >= int(maxFailoverCount) {
					glog.Warningf("%s/%s failure stores count reached the limit: %d", ns, tcName, tc.Spec.TiKV.MaxFailoverCount)
					return nil
				}
			}

			tc.Status.TiKV.FailureStores[storeID] = v1alpha1.TiKVFailureStore{
				PodName:   podName,
				StoreID:   store.ID,
				CreatedAt: metav1.Now(),
			}
		}
	}

	return nil
}

func (tf *tikvFailover) Recover(_ *v1alpha1.TidbCluster) {
	// Do nothing now
}

type fakeTiKVFailover struct{}

// NewFakeTiKVFailover returns a fake Failover
func NewFakeTiKVFailover() Failover {
	return &fakeTiKVFailover{}
}

func (ftf *fakeTiKVFailover) Failover(_ *v1alpha1.TidbCluster) error {
	return nil
}

func (ftf *fakeTiKVFailover) Recover(_ *v1alpha1.TidbCluster) {
	return
}
