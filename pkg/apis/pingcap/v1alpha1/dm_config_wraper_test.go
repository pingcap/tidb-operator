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
	"github.com/pingcap/tidb-operator/pkg/apis/util/toml"
)

func TestDMConfigWraper(t *testing.T) {
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
		var masterConfig MasterConfig
		f.Fuzz(&masterConfig)

		jsonData, err := json.Marshal(&masterConfig)
		t.Logf("case %d json:\n%s", i, string(jsonData))
		g.Expect(err).Should(BeNil())

		masterConfigWraper := NewMasterConfig()
		err = json.Unmarshal(jsonData, masterConfigWraper)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := masterConfigWraper.MarshalTOML()
		t.Logf("case %d toml back:\n%s", i, string(tomlDataBack))
		g.Expect(err).Should(BeNil())

		var masterConfigBack MasterConfig
		err = toml.Unmarshal(tomlDataBack, &masterConfigBack)
		g.Expect(err).Should(BeNil())
		if diff := cmp.Diff(masterConfig, masterConfigBack); diff != "" {
			t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			t.FailNow()
		}
	}
	for i := 0; i < 100; i++ {
		var workerConfig WorkerConfig
		f.Fuzz(&workerConfig)

		jsonData, err := json.Marshal(&workerConfig)
		t.Logf("case %d json:\n%s", i, string(jsonData))
		g.Expect(err).Should(BeNil())

		workerConfigWraper := NewWorkerConfig()
		err = json.Unmarshal(jsonData, workerConfigWraper)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := workerConfigWraper.MarshalTOML()
		t.Logf("case %d toml back:\n%s", i, string(tomlDataBack))
		g.Expect(err).Should(BeNil())

		var workerConfigBack WorkerConfig
		err = toml.Unmarshal(tomlDataBack, &workerConfigBack)
		g.Expect(err).Should(BeNil())
		if diff := cmp.Diff(workerConfig, workerConfigBack); diff != "" {
			t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			t.FailNow()
		}
	}
}
