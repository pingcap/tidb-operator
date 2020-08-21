// Copyright 2019 PingCAP, Inc.
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

package config

import (
	"testing"

	. "github.com/onsi/gomega"
)

type Simple struct {
	A string
	B int
}

func TestDeepCopyJsonObject(t *testing.T) {
	g := NewGomegaWithT(t)

	objects := []GenericConfig{
		New(nil),
		New(map[string]interface{}{
			"k1": true,
			"k2": "v2",
			"k3": 1,
			"k4": 'v',
			"k5": []byte("v5"),
			"k6": nil,
		}),
		New(map[string]interface{}{
			"k1": map[string]interface{}{
				"nest-1": map[string]interface{}{
					"nest-2": map[string]interface{}{
						"nest-3": "internal",
					},
				},
			},
		}),
		New(map[string]interface{}{
			"k1": map[string]interface{}{
				"nest-1": &Simple{
					A: "xx",
					B: 1,
				},
				"nest-2": Simple{
					A: "xx",
					B: 2,
				},
			},
		}),
	}

	for _, obj := range objects {
		copied := obj.DeepCopy()
		g.Expect(copied).To(Equal(&obj))

		out := New(nil)
		obj.DeepCopyInto(&out)
		g.Expect(out).To(Equal(obj))
	}
	copied := objects[1].DeepCopy()
	copied.Config["k1"] = false
	g.Expect(objects[1].Config["k1"]).To(Equal(true), "Mutation copy should net affect origin")
}
