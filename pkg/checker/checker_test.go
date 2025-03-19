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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestNeedRestartInstance(t *testing.T) {
	cases := []struct {
		desc string
		dbg  *v1alpha1.TiDBGroup
		db   *v1alpha1.TiDB
		need bool
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
			need: false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			res := NeedRestartInstance(c.dbg, c.db)
			assert.Equal(tt, c.need, res, c.desc)
		})
	}
}
