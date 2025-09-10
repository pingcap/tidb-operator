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

package adoption

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/reloadable"
)

func TestHash(t *testing.T) {
	cases := []struct {
		desc     string
		dbg      *v1alpha1.TiDBGroup
		db       *v1alpha1.TiDB
		expected bool
	}{
		{
			desc: "ignore template mode and keyspace",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Mode:     v1alpha1.TiDBModeNormal,
							Keyspace: "xxx",
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
						Mode:     v1alpha1.TiDBModeStandBy,
						Keyspace: "",
					},
				},
			},
			expected: true,
		},
		{
			desc: "ignore update stragegy",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							UpdateStrategy: v1alpha1.UpdateStrategy{
								Config: v1alpha1.ConfigUpdateStrategyHotReload,
							},
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
						UpdateStrategy: v1alpha1.UpdateStrategy{
							Config: v1alpha1.ConfigUpdateStrategyRestart,
						},
					},
				},
			},
			expected: true,
		},
		{
			desc: "ignore labels and annotations if mode changed from standby to normal",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						ObjectMeta: v1alpha1.ObjectMeta{
							Labels: map[string]string{
								"aaa": "bbb",
							},
							Annotations: map[string]string{
								"aaa": "bbb",
							},
						},
						Spec: v1alpha1.TiDBTemplateSpec{
							Mode:     v1alpha1.TiDBModeNormal,
							Keyspace: "xxx",
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"xxx": "yyy",
					},
					Annotations: map[string]string{
						"mmm": "nnn",
					},
				},
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
						Mode: v1alpha1.TiDBModeStandBy,
					},
				},
			},
			expected: true,
		},
		{
			desc: "don't ignore config",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Config: v1alpha1.ConfigFile(`aaa`),
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
						Config: v1alpha1.ConfigFile(`bbb`),
					},
				},
			},
			expected: false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			gh := hashTiDBGroup(c.dbg)
			ih := hashTiDB(c.db)
			if c.expected {
				assert.Equal(tt, gh, ih)
				assert.True(tt, reloadable.CheckTiDB(c.dbg, c.db), "tidb should be reloadable")
			} else {
				assert.NotEqual(tt, gh, ih)
			}
		})
	}
}
