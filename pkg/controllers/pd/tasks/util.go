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

package tasks

import (
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func PersistentVolumeClaimName(podName, volName string) string {
	// ref: https://github.com/pingcap/tidb-operator/blob/v1.6.0/pkg/apis/pingcap/v1alpha1/helpers.go#L92
	// NOTE: for v1, should use component as volName of data, e.g. pd
	return volName + "-" + podName
}

func LongestHealthPeer(pd *v1alpha1.PD, peers []*v1alpha1.PD) string {
	var p string
	lastTime := time.Now()
	for _, peer := range peers {
		if peer.Name == pd.Name {
			continue
		}
		cond := meta.FindStatusCondition(peer.Status.Conditions, v1alpha1.PDCondHealth)
		if cond == nil || cond.Status != metav1.ConditionTrue {
			continue
		}
		if cond.LastTransitionTime.Time.Before(lastTime) {
			lastTime = cond.LastTransitionTime.Time
			p = peer.Name
		}
	}

	return p
}
