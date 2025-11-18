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

package hasher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestHash(t *testing.T) {
	cases := []struct {
		desc     string
		obj      any
		expected string
	}{
		{
			desc:     "empty",
			obj:      nil,
			expected: "99c9b8c64",
		},
		{
			desc: "obj",
			obj: &v1alpha1.TiDBGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name: "xxx",
				},
			},
			// not important, feel free to change if tidb group is changed
			expected: "686bc9f6dd",
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			h := Hash(c.obj)
			assert.Equal(tt, c.expected, h)
		})
	}
}
