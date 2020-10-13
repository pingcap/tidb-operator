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

	"github.com/google/go-cmp/cmp"
	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
)

func TestPDConfigWraper(t *testing.T) {
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

		func(e *PDLabelPropertyConfig, c fuzz.Continue) {
			// do not generate nil value
			// when marshal will omit the kv, and unmarshal back will not equal like before without this kv.
			c.Fuzz(e)
			for k, v := range *e {
				if v == nil {
					delete(*e, k)
				}
			}
		},
	)
	for i := 0; i < 100; i++ {
		var pdConfig PDConfig
		f.Fuzz(&pdConfig)

		jsonData, err := json.Marshal(&pdConfig)
		t.Logf("case %d json:\n%s", i, string(jsonData))
		g.Expect(err).Should(BeNil())

		pdConfigWraper := NewPDConfig()
		err = json.Unmarshal(jsonData, pdConfigWraper)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := pdConfigWraper.MarshalTOML()
		t.Logf("case %d toml back:\n%s", i, string(tomlDataBack))
		g.Expect(err).Should(BeNil())

		var pdConfigBack PDConfig
		err = toml.Unmarshal(tomlDataBack, &pdConfigBack)
		g.Expect(err).Should(BeNil())
		if diff := cmp.Diff(pdConfig, pdConfigBack); diff != "" {
			t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			t.FailNow()
		}
	}
}
