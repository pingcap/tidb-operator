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
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type tikvFailover struct {
	tikvFailoverPeriod time.Duration
	recorder           record.EventRecorder
}

// NewTiKVFailover returns a tikv Failover
func NewTiKVFailover(tikvFailoverPeriod time.Duration, recorder record.EventRecorder) Failover {
	return &tikvFailover{tikvFailoverPeriod, recorder}
}

func (tf *tikvFailover) isPodDesired(tc *v1alpha1.TidbCluster, podName string) bool {
	ordinals := tc.TiKVStsDesiredOrdinals(true)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func (tf *tikvFailover) Failover(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for storeID, store := range tc.Status.TiKV.Stores {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		if !tf.isPodDesired(tc, podName) {
			// we should ignore the store record of deleted pod, otherwise the
			// record of deleted pod may be added back to failure stores
			// (before it enters into Offline/Tombstone state)
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
			if tc.Spec.TiKV.MaxFailoverCount != nil && *tc.Spec.TiKV.MaxFailoverCount > 0 {
				maxFailoverCount := *tc.Spec.TiKV.MaxFailoverCount
				if len(tc.Status.TiKV.FailureStores) >= int(maxFailoverCount) {
					klog.Warningf("%s/%s failure stores count reached the limit: %d", ns, tcName, tc.Spec.TiKV.MaxFailoverCount)
					return nil
				}
				tc.Status.TiKV.FailureStores[storeID] = v1alpha1.TiKVFailureStore{
					PodName:   podName,
					StoreID:   store.ID,
					CreatedAt: metav1.Now(),
				}
				msg := fmt.Sprintf("store[%s] is Down", store.ID)
				tf.recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "tikv", podName, msg))
			}
		}
	}

	return nil
}

func (tf *tikvFailover) Recover(tc *v1alpha1.TidbCluster) {
	for key, failureStore := range tc.Status.TiKV.FailureStores {
		if !tf.isPodDesired(tc, failureStore.PodName) {
			// If we delete the pods, e.g. by using advanced statefulset delete
			// slots feature. We should remove the record of undesired pods,
			// otherwise an extra replacement pod will be created.
			delete(tc.Status.TiKV.FailureStores, key)
		}
	}
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
