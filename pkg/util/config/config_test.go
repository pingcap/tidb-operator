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

	jsonpatch "github.com/evanphx/json-patch"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/json"
)

type Simple struct {
	A string
	B int
}

func TestGetSet(t *testing.T) {
	g := NewGomegaWithT(t)

	c := New(map[string]interface{}{})
	c.Set("a.b.c1", 1)
	c.Set("a.b.c2", 2)
	c.Set("a.b1", 1)
	c.Set("a.b2", 2)
	c.Set("a1", 1)
	c.Set("a2", 2)

	kv := map[string]int64{
		"a.b.c1": 1,
		"a.b.c2": 2,
		"a.b1":   1,
		"a.b2":   2,
		"a1":     1,
		"a2":     2,
	}

	for k, v := range kv {
		c.Set(k, v)
	}

	for k, v := range kv {
		g.Expect(c.Get(k).AsInt()).Should(Equal(v))
	}

	data, err := json.Marshal(c.Config)
	g.Expect(err).Should(BeNil())
	c.Config = make(map[string]interface{})
	err = json.Unmarshal(data, &c.Config)
	g.Expect(err).Should(BeNil())
	for k, v := range kv {
		g.Expect(c.Get(k).AsInt()).Should(Equal(v))
	}

	for i := 0; i < 100; i++ {
		c.Set("a.b1", i)
		get := c.Get("a.b1").AsInt()
		g.Expect(int(get)).Should(Equal(i))
	}
}

func fromJSON(data []byte) (c *GenericConfig, err error) {
	mp := make(map[string]interface{})
	err = json.Unmarshal(data, &mp)
	if err != nil {
		return nil, err
	}

	tmp := New(mp)
	return &tmp, nil
}

func TestJsonPatchDefaults(t *testing.T) {
	g := NewGomegaWithT(t)

	defaults := []byte(`{"name": "John", "age": 24, "height": 3.21, "a": {"a1": 1, "a2": 2}}`)
	custom := []byte(`{"name":"Jane", "a": {"a2": 22, "a3": 33}}`)
	merged := []byte(`{"a":{"a1":1,"a2":22,"a3":33},"age":24,"height":3.21,"name":"Jane"}`)

	defaultsMp := map[string]interface{}{}
	err := json.Unmarshal(defaults, &defaultsMp)
	g.Expect(err).Should(BeNil())

	customConfig, err := fromJSON(custom)
	g.Expect(err).Should(BeNil())
	mergedConfig, err := customConfig.JsonPatchDefaults(defaultsMp)
	g.Expect(err).Should(BeNil())

	mergedData, err := json.Marshal(mergedConfig.Config)
	g.Expect(err).Should(BeNil())
	eq := jsonpatch.Equal(mergedData, merged)
	g.Expect(eq).Should(BeTrue())
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
