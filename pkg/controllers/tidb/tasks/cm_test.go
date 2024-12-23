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
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v2"
)

func FakeContext(key types.NamespacedName, changes ...fake.ChangeFunc[ReconcileContext, *ReconcileContext]) *ReconcileContext {
	ctx := fake.Fake(changes...)
	ctx.Context = context.TODO()
	ctx.Key = key
	return ctx
}

func WithTiDB(tidb *v1alpha1.TiDB) fake.ChangeFunc[ReconcileContext, *ReconcileContext] {
	return func(obj *ReconcileContext) *ReconcileContext {
		obj.TiDB = tidb
		return obj
	}
}

func WithCluster(name string) fake.ChangeFunc[v1alpha1.TiDB, *v1alpha1.TiDB] {
	return func(tidb *v1alpha1.TiDB) *v1alpha1.TiDB {
		tidb.Spec.Cluster.Name = name
		return tidb
	}
}

func withStatusPDURL(pdURL string) fake.ChangeFunc[v1alpha1.Cluster, *v1alpha1.Cluster] {
	return func(cluster *v1alpha1.Cluster) *v1alpha1.Cluster {
		cluster.Status.PD = pdURL
		return cluster
	}
}

func withConfigForTiDB(config v1alpha1.ConfigFile) fake.ChangeFunc[v1alpha1.TiDB, *v1alpha1.TiDB] {
	return func(tidb *v1alpha1.TiDB) *v1alpha1.TiDB {
		tidb.Spec.Config = config
		return tidb
	}
}

func withSubdomain(subdomain string) fake.ChangeFunc[v1alpha1.TiDB, *v1alpha1.TiDB] {
	return func(tidb *v1alpha1.TiDB) *v1alpha1.TiDB {
		tidb.Spec.Subdomain = subdomain
		return tidb
	}
}

func TestConfigMap(t *testing.T) {
	cases := []struct {
		desc       string
		key        types.NamespacedName
		objs       []client.Object
		tidb       *v1alpha1.TiDB
		cluster    *v1alpha1.Cluster
		expected   task.Result
		expectedCM *corev1.ConfigMap
	}{
		{
			desc: "cm doesn't exist",
			key: types.NamespacedName{
				Name: "test-tidb",
			},
			objs: []client.Object{},
			tidb: fake.FakeObj(
				"test-tidb",
				WithCluster("test-cluster"),
				withConfigForTiDB(v1alpha1.ConfigFile("")),
				withSubdomain("subdomain"),
				fake.Label[v1alpha1.TiDB]("aaa", "bbb"),
				fake.UID[v1alpha1.TiDB]("test-uid"),
				fake.Label[v1alpha1.TiDB](v1alpha1.LabelKeyInstanceRevisionHash, "foo"),
			),
			cluster: fake.FakeObj("test-cluster",
				withStatusPDURL("http://test-pd.default:2379"),
			),
			expected: task.Complete().With(""),
			expectedCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-tidb-tidb",
					Labels: map[string]string{
						"aaa":                                 "bbb",
						v1alpha1.LabelKeyInstance:             "test-tidb",
						v1alpha1.LabelKeyInstanceRevisionHash: "foo",
						v1alpha1.LabelKeyConfigHash:           "7d6fc488b7",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         v1alpha1.SchemeGroupVersion.String(),
							Kind:               "TiDB",
							Name:               "test-tidb",
							UID:                "test-uid",
							BlockOwnerDeletion: ptr.To(true),
							Controller:         ptr.To(true),
						},
					},
				},
				Data: map[string]string{
					v1alpha1.ConfigFileName: `advertise-address = 'test-tidb-tidb.subdomain.default.svc'
graceful-wait-before-shutdown = 60
host = '::'
path = 'test-pd.default:2379'
store = 'tikv'

[log]
slow-query-file = '/var/log/tidb/slowlog'
`,
				},
			},
		},
		{
			desc: "cm exists, update cm",
			key: types.NamespacedName{
				Name: "test-tidb",
			},
			objs: []client.Object{
				fake.FakeObj[corev1.ConfigMap](
					"test-tidb",
				),
			},
			tidb: fake.FakeObj(
				"test-tidb",
				WithCluster("test-cluster"),
				withConfigForTiDB(`zzz = 'zzz'
graceful-wait-before-shutdown = 60`),
				withSubdomain("subdomain"),
				fake.Label[v1alpha1.TiDB]("aaa", "bbb"),
				fake.UID[v1alpha1.TiDB]("test-uid"),
				fake.Label[v1alpha1.TiDB](v1alpha1.LabelKeyInstanceRevisionHash, "foo"),
			),
			cluster: fake.FakeObj("test-cluster",
				withStatusPDURL("http://test-pd.default:2379"),
			),
			expected: task.Complete().With(""),
			expectedCM: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-tidb-tidb",
					Labels: map[string]string{
						"aaa":                                 "bbb",
						v1alpha1.LabelKeyInstance:             "test-tidb",
						v1alpha1.LabelKeyInstanceRevisionHash: "foo",
						v1alpha1.LabelKeyConfigHash:           "7bd44dcc66",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         v1alpha1.SchemeGroupVersion.String(),
							Kind:               "TiDB",
							Name:               "test-tidb",
							UID:                "test-uid",
							BlockOwnerDeletion: ptr.To(true),
							Controller:         ptr.To(true),
						},
					},
				},
				Data: map[string]string{
					v1alpha1.ConfigFileName: `advertise-address = 'test-tidb-tidb.subdomain.default.svc'
graceful-wait-before-shutdown = 60
host = '::'
path = 'test-pd.default:2379'
store = 'tikv'
zzz = 'zzz'

[log]
slow-query-file = '/var/log/tidb/slowlog'
`,
				},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			// append TiDB into store
			objs := c.objs
			objs = append(objs, c.tidb)

			ctx := FakeContext(c.key, WithTiDB(c.tidb))
			ctx.Cluster = c.cluster
			fc := client.NewFakeClient(objs...)
			tk := NewTaskConfigMap(logr.Discard(), fc)
			res := tk.Sync(ctx)

			assert.Equal(tt, c.expected.Status(), res.Status(), res.Message())
			assert.Equal(tt, c.expected.RequeueAfter(), res.RequeueAfter(), res.Message())
			// Ignore message assertion
			// TODO: maybe assert the message format?

			if res.Status() == task.SFail {
				return
			}

			cm := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: ConfigMapName(ctx.TiDB.PodName()),
				},
			}
			err := fc.Get(ctx, client.ObjectKeyFromObject(&cm), &cm)
			require.NoError(tt, err)

			// reset cm gvk and managed fields
			cm.APIVersion = ""
			cm.Kind = ""
			cm.SetManagedFields(nil)
			assert.Equal(tt, c.expectedCM, &cm)
		})
	}
}
