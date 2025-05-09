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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func TestFeatureGates(t *testing.T) {
	const (
		uid1 = "aaa"
		uid2 = "bbb"

		featA = "aaa"
		featB = "bbb"
	)

	type controller struct {
		desc    string
		feat    string
		enabled bool

		useCluster        *v1alpha1.Cluster
		useBeforeRegister bool
		skipUse           bool

		failToUse bool
	}
	cases := []struct {
		desc    string
		cluster *v1alpha1.Cluster

		deregister  bool
		controllers []controller
	}{
		{
			desc:    "aaa is enabled",
			cluster: fakeCluster("", "aaa", 1, uid1, featA),

			controllers: []controller{
				{
					desc:              "not init",
					useBeforeRegister: true,
					failToUse:         true,
				},
				{
					desc:    "inited",
					feat:    featA,
					enabled: true,
				},
				{
					desc:    "bbb not enabled",
					feat:    featB,
					enabled: false,
				},
			},
		},
		{
			desc:    "in another cluster, bbb is enabled",
			cluster: fakeCluster("xxx", "bbb", 1, uid1, featB),

			controllers: []controller{
				{
					desc:              "not init",
					useBeforeRegister: true,
					failToUse:         true,
				},
				{
					desc:    "inited",
					feat:    featB,
					enabled: true,
				},
				{
					desc:    "aaa not enabled",
					feat:    featA,
					enabled: false,
				},
			},
		},
		{
			desc:    "feature gates are not changed, reuse",
			cluster: fakeCluster("", "aaa", 1, uid1, featA),

			controllers: []controller{
				{
					desc:              "before register",
					useBeforeRegister: true,
					feat:              featA,
					enabled:           true,
				},
				{
					desc:    "inited",
					feat:    featA,
					enabled: true,
				},
				{
					desc:    "bbb not enabled",
					feat:    featB,
					enabled: false,
				},
			},
		},
		{
			desc:    "feature gates are changed",
			cluster: fakeCluster("", "aaa", 2, uid1, featB),

			controllers: []controller{
				{
					desc:              "before register, feature gates are outdated",
					useBeforeRegister: true,
					failToUse:         true,
				},
				{
					desc:              "still using legacy cluster",
					useBeforeRegister: true,
					useCluster:        fakeCluster("", "aaa", 1, uid1, featA),
					feat:              featA,
					enabled:           true,
				},
				{
					desc:      "cannot use because of the outdated feature gates are still in use",
					failToUse: true,
				},
				{
					desc:    "not use, still use outdated feature gates, aaa is enabled",
					skipUse: true,
					feat:    featA,
					enabled: true,
				},
				{
					desc:    "not use, still use outdated feature gates, bbb is disabled",
					skipUse: true,
					feat:    featB,
					enabled: false,
				},
			},
		},
		{
			desc:    "ok to use",
			cluster: fakeCluster("", "aaa", 2, uid1, featB),

			controllers: []controller{
				{
					desc:              "before register, ok to use",
					useBeforeRegister: true,
					feat:              featB,
					enabled:           true,
				},
				{
					desc:    "after register, ok to use",
					feat:    featB,
					enabled: true,
				},
			},
		},
		{
			desc:    "uid and generation are changed but feature gates are not changed",
			cluster: fakeCluster("", "aaa", 3, uid2, featB),

			controllers: []controller{
				{
					desc:              "before register, ok to use",
					useBeforeRegister: true,
					feat:              featB,
					enabled:           true,
				},
				{
					desc:    "after register, ok to use",
					feat:    featB,
					enabled: true,
				},
			},
		},
		{
			desc:       "deregister is worked",
			cluster:    fakeCluster("", "aaa", 3, uid2, featB),
			deregister: true,

			controllers: []controller{
				{
					desc:              "before deregister, still ok to use",
					useBeforeRegister: true,
					feat:              featB,
					enabled:           true,
				},
				{
					desc:      "after deregister, fail to use",
					failToUse: true,
					feat:      featB,
				},
			},
		},
	}

	fg := NewFeatureGates()
	format := "%s: failed ctrl: %s"

	for i := range cases {
		c := &cases[i]
		defer func() {
			recover()
		}()

		func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for _, ctrl := range c.controllers {
				if ctrl.useBeforeRegister && !ctrl.skipUse {
					cluster := c.cluster
					if ctrl.useCluster != nil {
						cluster = ctrl.useCluster
					}
					err := fg.Use(ctx, cluster)
					if ctrl.failToUse {
						require.Error(t, err, format, c.desc, ctrl.desc)
						fmt.Println("fail to use:", "desc:", c.desc, "ctrl:", ctrl.desc, "err:", err)
					} else {
						require.NoError(t, err, format, c.desc, ctrl.desc)
					}
				}
			}
			if c.deregister {
				fg.Deregister(c.cluster.Namespace, c.cluster.Name)
				for _, ctrl := range c.controllers {
					if ctrl.useBeforeRegister || ctrl.skipUse {
						continue
					}
					cluster := c.cluster
					if ctrl.useCluster != nil {
						cluster = ctrl.useCluster
					}
					require.Error(t, fg.Use(ctx, cluster), format, c.desc, ctrl.desc)
					assert.Panics(t, func() {
						fg.Enabled(c.cluster.Namespace, c.cluster.Name, meta.Feature(ctrl.feat))
					}, format, c.desc, ctrl.desc)
				}

				return
			}

			fg.Register(c.cluster)

			for _, ctrl := range c.controllers {
				if !ctrl.useBeforeRegister && !ctrl.skipUse {
					cluster := c.cluster
					if ctrl.useCluster != nil {
						cluster = ctrl.useCluster
					}
					err := fg.Use(ctx, cluster)
					if ctrl.failToUse {
						require.Error(t, err, format, c.desc, ctrl.desc)
						fmt.Println("fail to use after:", "desc:", c.desc, "ctrl:", ctrl.desc, "err:", err)
					} else {
						require.NoError(t, err, format, c.desc, ctrl.desc)
					}
				}
				if ctrl.failToUse {
					continue
				}
				enabled := fg.Enabled(c.cluster.Namespace, c.cluster.Name, meta.Feature(ctrl.feat))
				assert.Equal(t, ctrl.enabled, enabled, format, c.desc, ctrl.desc)
			}

			panic("panic to check dead lock")
		}()
		// wait an arbitrary time to make sure all `unuse` are finished
		time.Sleep(time.Millisecond)
	}
}

func fakeCluster(ns, name string, generation int64, uid types.UID, features ...string) *v1alpha1.Cluster {
	var fs []meta.FeatureGate
	for _, feat := range features {
		fs = append(fs, meta.FeatureGate{
			Name: meta.Feature(feat),
		})
	}
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  ns,
			Generation: generation,
			UID:        uid,
		},
		Spec: v1alpha1.ClusterSpec{
			FeatureGates: fs,
		},
	}
}
