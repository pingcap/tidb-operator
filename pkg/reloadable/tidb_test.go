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

package reloadable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestCheckTiDB(t *testing.T) {
	cases := []struct {
		desc       string
		dbg        *v1alpha1.TiDBGroup
		db         *v1alpha1.TiDB
		reloadable bool
	}{
		{
			desc: "config is changed and policy is from restart to hot reload",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Config: v1alpha1.ConfigFile("xxx"),
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
						Config: v1alpha1.ConfigFile("yyy"),
						UpdateStrategy: v1alpha1.UpdateStrategy{
							Config: v1alpha1.ConfigUpdateStrategyRestart,
						},
					},
				},
			},
			reloadable: true,
		},
		{
			desc: "config is changed and policy is from hot reload to restart",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Config: v1alpha1.ConfigFile("xxx"),
							UpdateStrategy: v1alpha1.UpdateStrategy{
								Config: v1alpha1.ConfigUpdateStrategyRestart,
							},
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
						Config: v1alpha1.ConfigFile("yyy"),
						UpdateStrategy: v1alpha1.UpdateStrategy{
							Config: v1alpha1.ConfigUpdateStrategyHotReload,
						},
					},
				},
			},
			reloadable: false,
		},
		{
			desc: "config is changed and policy is restart",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Config: v1alpha1.ConfigFile("xxx"),
							UpdateStrategy: v1alpha1.UpdateStrategy{
								Config: v1alpha1.ConfigUpdateStrategyRestart,
							},
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
						Config: v1alpha1.ConfigFile("yyy"),
						UpdateStrategy: v1alpha1.UpdateStrategy{
							Config: v1alpha1.ConfigUpdateStrategyRestart,
						},
					},
				},
			},
			reloadable: false,
		},
		{
			desc: "config is changed and policy is hot reload",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Config: v1alpha1.ConfigFile("xxx"),
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
						Config: v1alpha1.ConfigFile("yyy"),
						UpdateStrategy: v1alpha1.UpdateStrategy{
							Config: v1alpha1.ConfigUpdateStrategyHotReload,
						},
					},
				},
			},
			reloadable: true,
		},
		{
			desc: "add pod labels overlay",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Overlay: &v1alpha1.Overlay{
								Pod: &v1alpha1.PodOverlay{
									ObjectMeta: v1alpha1.ObjectMeta{
										Labels: map[string]string{"test": "test"},
									},
								},
							},
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{
						Overlay: nil,
					},
				},
			},
			reloadable: true,
		},
		{
			desc: "add a restart annotation",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						ObjectMeta: v1alpha1.ObjectMeta{
							Annotations: map[string]string{
								"foo": "bar",
							},
						},
						Spec: v1alpha1.TiDBTemplateSpec{},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{},
				},
			},
			reloadable: false,
		},
		{
			desc: "update the restart annotation",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						ObjectMeta: v1alpha1.ObjectMeta{
							Annotations: map[string]string{
								"foo": "bar",
							},
						},
						Spec: v1alpha1.TiDBTemplateSpec{},
					},
				},
			},
			db: &v1alpha1.TiDB{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"foo": "zoo",
					},
				},
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{},
				},
			},
			reloadable: false,
		},
		{
			desc: "update the server labels",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						Spec: v1alpha1.TiDBTemplateSpec{
							Server: v1alpha1.TiDBServer{
								Labels: map[string]string{"foo": "bar"},
							},
						},
					},
				},
			},
			db: &v1alpha1.TiDB{
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{},
				},
			},
			reloadable: true,
		},
		{
			desc: "change labels and annotations is not reloadable",
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
						Spec: v1alpha1.TiDBTemplateSpec{},
					},
				},
			},
			db: &v1alpha1.TiDB{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"xxx": "yyy",
					},
					Annotations: map[string]string{
						"xxx": "yyy",
					},
				},
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{},
				},
			},
			reloadable: false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			res := CheckTiDB(c.dbg, c.db)
			assert.Equal(tt, c.reloadable, res, c.desc)
		})
	}
}
