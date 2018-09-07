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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

type tikvFailover struct {
	pdControl controller.PDControlInterface
}

// NewTiKVFailover returns a tikv Failover
func NewTiKVFailover(pdControl controller.PDControlInterface) Failover {
	return &tikvFailover{pdControl}
}

func (tf *tikvFailover) Failover(tc *v1alpha1.TidbCluster) error {
	cfg, err := tf.pdControl.GetPDClient(tc).GetConfig()
	if err != nil {
		return err
	}

	for storeID, store := range tc.Status.TiKV.Stores {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		deadline := store.LastTransitionTime.Add(cfg.Schedule.MaxStoreDownTime.Duration)
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
			tc.Status.TiKV.FailureStores[storeID] = v1alpha1.TiKVFailureStore{
				PodName:  podName,
				StoreID:  store.ID,
				Replicas: tc.Spec.TiKV.Replicas,
			}
			tc.Spec.TiKV.Replicas++
		}
	}

	return nil
}

func (tf *tikvFailover) Recover(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.FailureStores = nil
}

func allTiKVStoressAreReady(tc *v1alpha1.TidbCluster) bool {
	if int(tc.Spec.TiKV.Replicas) != len(tc.Status.TiKV.Stores) {
		return false
	}
	for _, store := range tc.Status.TiKV.Stores {
		if store.State != v1alpha1.TiKVStateUp {
			return false
		}
	}
	return true
}

type fakeTiKVFailover struct{}

// NewFakeTiKVFailover returns a fake Failover
func NewFakeTiKVFailover() Failover {
	return &fakeTiKVFailover{}
}

func (ftf *fakeTiKVFailover) Failover(tc *v1alpha1.TidbCluster) error {
	return nil
}

func (ftf *fakeTiKVFailover) Recover(tc *v1alpha1.TidbCluster) {
	return
}
