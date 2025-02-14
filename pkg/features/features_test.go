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

package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func TestFeatureGates(t *testing.T) {
	const (
		uid1 = "aaa"
		uid2 = "bbb"
	)
	cases := []struct {
		desc    string
		cluster *v1alpha1.Cluster

		deregister   bool
		failToVerify bool
		feat         meta.Feature
		enabled      bool
	}{
		{
			desc: "aaa is enabled",
			cluster: fakeCluster("", "aaa", 1, uid1, []meta.FeatureGate{
				{
					Name: meta.Feature("aaa"),
				},
			}),

			// no cluster is registered
			failToVerify: true,
			feat:         meta.Feature("aaa"),
			enabled:      true,
		},
		{
			desc: "bbb is not enabled",
			cluster: fakeCluster("", "aaa", 1, uid1, []meta.FeatureGate{
				{
					Name: meta.Feature("aaa"),
				},
			}),

			feat:    meta.Feature("bbb"),
			enabled: false,
		},
		{
			desc: "in another cluster, bbb is enabled",
			cluster: fakeCluster("xxx", "bbb", 1, uid1, []meta.FeatureGate{
				{
					Name: meta.Feature("bbb"),
				},
			}),

			failToVerify: true,
			feat:         meta.Feature("bbb"),
			enabled:      true,
		},
		{
			desc: "enable bbb and disable aaa",
			cluster: fakeCluster("", "aaa", 2, uid1, []meta.FeatureGate{
				{
					Name: meta.Feature("bbb"),
				},
			}),

			// cluster is changed
			failToVerify: true,
			feat:         meta.Feature("bbb"),
			enabled:      true,
		},
		{
			desc: "enable bbb and disable aaa",
			cluster: fakeCluster("", "aaa", 2, uid1, []meta.FeatureGate{
				{
					Name: meta.Feature("bbb"),
				},
			}),

			feat:    meta.Feature("aaa"),
			enabled: false,
		},
		{
			// not happen actually
			desc: "cluster uid and generation are not changed, only test cache is worked, aaa is still disabled",
			cluster: fakeCluster("", "aaa", 2, uid1, []meta.FeatureGate{
				{
					Name: meta.Feature("aaa"),
				},
			}),

			feat:    meta.Feature("aaa"),
			enabled: false,
		},
		{
			// not happen actually
			desc: "cluster uid and generation are not changed, only test cache is worked, bbb is still enabled",
			cluster: fakeCluster("", "aaa", 2, uid1, []meta.FeatureGate{
				{
					Name: meta.Feature("aaa"),
				},
			}),

			feat:    meta.Feature("bbb"),
			enabled: true,
		},
		{
			desc: "cluster uid is changed",
			cluster: fakeCluster("", "aaa", 2, uid2, []meta.FeatureGate{
				{
					Name: meta.Feature("aaa"),
				},
			}),

			// cluster's uid is changed
			failToVerify: true,
			feat:         meta.Feature("aaa"),
			enabled:      true,
		},
		{
			desc: "deregister is worked",
			cluster: fakeCluster("", "aaa", 2, uid2, []meta.FeatureGate{
				{
					Name: meta.Feature("bbb"),
				},
			}),
			deregister: true,

			feat:    meta.Feature("bbb"),
			enabled: true,
		},
	}

	fg := NewFeatureGates()
	for i := range cases {
		c := &cases[i]

		err := fg.Verify(c.cluster)
		if c.failToVerify {
			assert.Error(t, err, c.desc)
		} else {
			assert.NoError(t, err, c.desc)
		}

		if c.deregister {
			fg.Deregister(c.cluster.Namespace, c.cluster.Name)
			assert.Error(t, fg.Verify(c.cluster), c.desc)
			assert.Panics(t, func() {
				fg.Enabled(c.cluster.Namespace, c.cluster.Name, c.feat)
			}, c.desc)
		}
		fg.Register(c.cluster)
		enabled := fg.Enabled(c.cluster.Namespace, c.cluster.Name, c.feat)
		assert.Equal(t, c.enabled, enabled, c.desc)
	}
}

func fakeCluster(ns, name string, generation int64, uid types.UID, features []meta.FeatureGate) *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  ns,
			Generation: generation,
			UID:        uid,
		},
		Spec: v1alpha1.ClusterSpec{
			FeatureGates: features,
		},
	}
}
