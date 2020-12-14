// Copyright 2020 PingCAP, Inc.
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

package v1alpha1

import (
	"encoding/json"
	"strconv"
	"testing"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
)

func TestTiDBConfigWraper(t *testing.T) {
	g := NewGomegaWithT(t)

	f := fuzz.New().Funcs(
		// the toml pkg we use do support to unmarshal a value overflow int64
		// when unmarshal to go map[string]interface{
		// just set the value in uint32 range now for
		func(e *uint64, c fuzz.Continue) {
			*e = uint64(c.Uint32())
		},
		func(e *uint, c fuzz.Continue) {
			*e = uint(c.Uint32())
		},

		func(e *string, c fuzz.Continue) {
			*e = "s" + strconv.Itoa(c.Intn(100))
		},
	)
	for i := 0; i < 100; i++ {
		var tidbConfig TiDBConfig
		f.Fuzz(&tidbConfig)

		jsonData, err := json.Marshal(&tidbConfig)
		g.Expect(err).Should(BeNil())

		tidbConfigWraper := NewTiDBConfig()
		err = json.Unmarshal(jsonData, tidbConfigWraper)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := tidbConfigWraper.MarshalTOML()
		g.Expect(err).Should(BeNil())

		var tidbConfigBack TiDBConfig
		err = toml.Unmarshal(tomlDataBack, &tidbConfigBack)
		g.Expect(err).Should(BeNil())
		g.Expect(tidbConfigBack).Should(Equal(tidbConfig))
	}
}

func TestUnmarshlJSON(t *testing.T) {
	g := NewGomegaWithT(t)

	type Tmp struct {
		Float float64 `toml:"float"`
		Int   int     `toml:"int"`
	}

	data := `{"float": 1, "int": 1}`

	tmp := new(Tmp)
	config, err := unmarshalJSON([]byte(data), tmp)
	g.Expect(err).Should(BeNil())

	g.Expect(config.Get("float").MustFloat()).Should(Equal(1.0))
	g.Expect(config.Get("int").MustInt()).Should(Equal(int64(1)))
}
