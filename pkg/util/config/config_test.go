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

	kv := map[string]int64{
		"a.b.c1": 1,
		"a.b.c2": 2,
		"a.b1":   1,
		"a.b2":   2,
		"a1":     1,
		"a2":     2,
	}

	c := New(map[string]interface{}{})

	for k, v := range kv {
		c.Set(k, v)
	}

	for k, v := range kv {
		g.Expect(c.Get(k).AsInt()).Should(Equal(v))
	}

	data, err := json.Marshal(c.mp)
	g.Expect(err).Should(BeNil())
	c.mp = make(map[string]interface{})
	err = json.Unmarshal(data, &c.mp)
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
	return tmp, nil
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

	mergedData, err := json.Marshal(mergedConfig.mp)
	g.Expect(err).Should(BeNil())
	eq := jsonpatch.Equal(mergedData, merged)
	g.Expect(eq).Should(BeTrue())
}

func TestDeepCopyJsonObject(t *testing.T) {
	g := NewGomegaWithT(t)

	objects := []*GenericConfig{
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
		g.Expect(copied).To(Equal(obj))

		out := New(nil)
		obj.DeepCopyInto(out)
		g.Expect(out).To(Equal(obj))
	}
	copied := objects[1].DeepCopy()
	copied.mp["k1"] = false
	g.Expect(objects[1].mp["k1"]).To(Equal(true), "Mutation copy should net affect origin")
}

func TestJsonMarshal(t *testing.T) {
	type S struct {
		Config *GenericConfig `json:"config,omitempty"`
	}

	g := NewGomegaWithT(t)

	// test Config should be nil
	s := new(S)
	err := json.Unmarshal([]byte("{}"), s)
	g.Expect(err).Should(BeNil())
	g.Expect(s.Config).Should(BeNil())
	data, err := json.Marshal(s)
	g.Expect(err).Should(BeNil())
	g.Expect(err).Should(BeNil())
	g.Expect(s.Config).Should(BeNil())

	// test Config should not be nil
	s = new(S)
	err = json.Unmarshal([]byte("{\"config\":{}}"), s)
	g.Expect(err).Should(BeNil())
	g.Expect(s.Config).ShouldNot(BeNil())
	data, err = json.Marshal(s)
	g.Expect(err).Should(BeNil())
	s = new(S)
	err = json.Unmarshal(data, s)
	g.Expect(err).Should(BeNil())
	g.Expect(s.Config).ShouldNot(BeNil())

	// test keep int or float type
	s = new(S)
	s.Config = New(map[string]interface{}{})
	s.Config.Set("int", 1)
	s.Config.Set("float", float64(1.0))
	data, err = json.Marshal(s)
	t.Log("data: ", string(data))
	g.Expect(err).Should(BeNil())
	s = new(S)
	err = json.Unmarshal(data, s)
	g.Expect(err).Should(BeNil())
	integer := s.Config.Get("int").AsInt()
	g.Expect(integer).Should(Equal(int64(1)))
	float := s.Config.Get("float").AsFloat()
	g.Expect(float).Should(Equal(float64(1.0)))

}
