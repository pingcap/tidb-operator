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
	"strconv"
	"testing"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
)

func TestTiKVConfigWraper(t *testing.T) {
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
		var tikvConfig TiKVConfig
		f.Fuzz(&tikvConfig)

		tomlData, err := toml.Marshal(&tikvConfig)
		g.Expect(err).Should(BeNil())

		tikvConfigWraper := NewTiKVConfig()
		err = tikvConfigWraper.UnmarshalTOML(tomlData)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := tikvConfigWraper.MarshalTOML()
		g.Expect(err).Should(BeNil())

		var tikvConfigBack TiKVConfig
		err = toml.Unmarshal(tomlDataBack, &tikvConfigBack)
		g.Expect(err).Should(BeNil())
		g.Expect(tikvConfigBack).Should(Equal(tikvConfig))
	}
}
