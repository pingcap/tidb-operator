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
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/json"
)

type Simple struct {
	A string
	B int
}

func TestStringSlice(t *testing.T) {
	g := NewGomegaWithT(t)
	data := `slice = ["s1", "s2"]
	not_slice = "ok"`

	c := New(map[string]interface{}{})
	err := c.UnmarshalTOML([]byte(data))
	g.Expect(err).Should(BeNil())

	v := c.Get("slice") // unmarshal from toml will be []interface{}
	g.Expect(v).ShouldNot(BeNil())
	slice, err := v.AsStringSlice()
	g.Expect(err).Should(BeNil())
	g.Expect(slice).Should(Equal([]string{"s1", "s2"}))

	// test by set as []string
	c.Set("slice", []string{"s1"})
	v = c.Get("slice")
	g.Expect(v).ShouldNot(BeNil())
	slice, err = v.AsStringSlice()
	g.Expect(err).Should(BeNil())
	g.Expect(slice).Should(Equal([]string{"s1"}))

	// test negative
	v = c.Get("not_slice")
	g.Expect(v).ShouldNot(BeNil())
	_, err = v.AsStringSlice()
	g.Expect(err).ShouldNot(BeNil())
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
		c.Set("f-"+k, float64(v))
		c.Set("s-"+k, strconv.FormatInt(v, 10))
	}

	for k, v := range kv {
		g.Expect(c.Get(k).MustInt()).Should(Equal(v))
		g.Expect(c.Get("f-" + k).MustFloat()).Should(Equal(float64(v)))
		g.Expect(c.Get("s-" + k).MustString()).Should(Equal(strconv.Itoa(int(v))))
	}

	// test overwrite the same key
	k := "a.b1"
	for v := int64(0); v < 100; v++ {
		c.Set(k, v)
		g.Expect(c.Get(k).MustInt()).Should(Equal(v))

		c.Set("f-"+k, float64(v))
		g.Expect(c.Get("f-" + k).MustFloat()).Should(Equal(float64(v)))

		c.Set("s-"+k, strconv.FormatInt(v, 10))
		g.Expect(c.Get("s-" + k).MustString()).Should(Equal(strconv.Itoa(int(v))))
	}
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
	copied.Inner()["k1"] = false
	g.Expect(objects[1].Inner()["k1"]).To(Equal(true), "Mutation copy should net affect origin")
}

func TestMarshalTOML(t *testing.T) {
	g := NewGomegaWithT(t)

	c := New(map[string]interface{}{
		"int":       int64(1),
		"float":     1.0,
		"str":       "str",
		"str_slice": []interface{}{"s1", "s2"},
	})

	data, err := c.MarshalTOML()
	g.Expect(err).Should(BeNil())
	t.Log("toml: ", string(data))

	cback := New(nil)
	err = cback.UnmarshalTOML(data)
	g.Expect(err).Should(BeNil())
	if diff := cmp.Diff(c, cback); diff != "" {
		t.Errorf("(-want,+got): %s\n", diff)
	}
}

func TestMarshlJSON(t *testing.T) {
	type S struct {
		Config *GenericConfig `json:"config,omitempty"`
	}

	g := NewGomegaWithT(t)

	s := &S{
		Config: New(map[string]interface{}{}),
	}
	s.Config.Set("sk", "v")
	s.Config.Set("ik", int64(1))

	data, err := json.Marshal(s)
	g.Expect(err).Should(BeNil())

	sback := new(S)
	err = json.Unmarshal(data, sback)
	g.Expect(err).Should(BeNil())
	g.Expect(sback).Should(Equal(s))
}

func TestJsonOmitempty(t *testing.T) {
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
	s = new(S)
	err = json.Unmarshal(data, s)
	g.Expect(err).Should(BeNil())
	g.Expect(s.Config).Should(BeNil())

	// test Config should not be nil
	s = new(S)
	err = json.Unmarshal([]byte("{\"config\":\"a = 1\"}"), s)
	g.Expect(err).Should(BeNil())
	g.Expect(s.Config).ShouldNot(BeNil())
	data, err = json.Marshal(s)
	g.Expect(err).Should(BeNil())
	s = new(S)
	err = json.Unmarshal(data, s)
	g.Expect(err).Should(BeNil())
	g.Expect(s.Config).ShouldNot(BeNil())
}
