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
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func TestCheckTiDB(t *testing.T) {
	cases := []struct {
		desc       string
		dbg        *v1alpha1.TiDBGroup
		db         *v1alpha1.TiDB
		reloadable bool
	}{
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
								metav1alpha1.RestartAnnotationKey: "foo",
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
			need: true,
		},
		{
			desc: "update the restart annotation",
			dbg: &v1alpha1.TiDBGroup{
				Spec: v1alpha1.TiDBGroupSpec{
					Template: v1alpha1.TiDBTemplate{
						ObjectMeta: v1alpha1.ObjectMeta{
							Annotations: map[string]string{
								metav1alpha1.RestartAnnotationKey: "foo",
							},
						},
						Spec: v1alpha1.TiDBTemplateSpec{},
					},
				},
			},
			db: &v1alpha1.TiDB{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						metav1alpha1.RestartAnnotationKey: "bar",
					},
				},
				Spec: v1alpha1.TiDBSpec{
					TiDBTemplateSpec: v1alpha1.TiDBTemplateSpec{},
				},
			},
			need: true,
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
