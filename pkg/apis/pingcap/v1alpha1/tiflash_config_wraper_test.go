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
	"sigs.k8s.io/yaml"
)

func TestTiFlashCommonConfigWraper(t *testing.T) {
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
		var config CommonConfig
		f.Fuzz(&config)

		jsonData, err := json.Marshal(&config)
		t.Logf("case %d json:\n%s", i, string(jsonData))
		g.Expect(err).Should(BeNil())

		configWraper := NewTiFlashCommonConfig()
		err = json.Unmarshal(jsonData, configWraper)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := configWraper.MarshalTOML()
		t.Logf("case %d toml back:\n%s", i, string(tomlDataBack))
		g.Expect(err).Should(BeNil())

		var configBack CommonConfig
		err = toml.Unmarshal(tomlDataBack, &configBack)
		g.Expect(err).Should(BeNil())
		if diff := cmp.Diff(config, configBack); diff != "" {
			t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			t.FailNow()
		}
	}
}

func TestTiFlashProxyConfigWraper(t *testing.T) {
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
		var config ProxyConfig
		f.Fuzz(&config)

		jsonData, err := json.Marshal(&config)
		t.Logf("case %d json:\n%s", i, string(jsonData))
		g.Expect(err).Should(BeNil())

		configWraper := NewTiFlashProxyConfig()
		err = json.Unmarshal(jsonData, configWraper)
		g.Expect(err).Should(BeNil())

		tomlDataBack, err := configWraper.MarshalTOML()
		t.Logf("case %d toml back:\n%s", i, string(tomlDataBack))
		g.Expect(err).Should(BeNil())

		var configBack ProxyConfig
		err = toml.Unmarshal(tomlDataBack, &configBack)
		g.Expect(err).Should(BeNil())
		if diff := cmp.Diff(config, configBack); diff != "" {
			t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			t.FailNow()
		}
	}
}

func TestTiFlashConfigToml(t *testing.T) {
	var data = []byte(`tiflash:
  config:
    config: |
      [logger]
        level = "warn"
        count = 9
    proxy: |
      log-level = "warn"
      [gc]
        batch-keys = 501
`)

	g := NewGomegaWithT(t)
	var err error
	data, err = yaml.YAMLToJSON(data)
	g.Expect(err).Should(BeNil())

	tc := &TidbClusterSpec{}
	err = json.Unmarshal(data, tc)
	g.Expect(err).Should(BeNil())

	config := tc.TiFlash.Config
	g.Expect(config.Common.Get("logger.level").MustString()).Should(Equal("warn"))
	g.Expect(config.Common.Get("logger.count").MustInt()).Should(Equal(int64(9)))
	g.Expect(config.Proxy.Get("log-level").MustString()).Should(Equal("warn"))
	g.Expect(config.Proxy.Get("gc.batch-keys").MustInt()).Should(Equal(int64(501)))
}

func TestTiFlashConfigJson(t *testing.T) {
	var data = []byte(`tiflash:
  config:
    config:
      logger:
        level: warn
        count: 9
    proxy:
      log-level: warn
      gc:
        batch-keys: 501
`)

	t.Log(string(data))

	g := NewGomegaWithT(t)
	var err error
	data, err = yaml.YAMLToJSON(data)
	g.Expect(err).Should(BeNil())

	tc := &TidbClusterSpec{}
	err = json.Unmarshal(data, tc)
	g.Expect(err).Should(BeNil())

	config := tc.TiFlash.Config
	g.Expect(config.Common.Get("logger.level").MustString()).Should(Equal("warn"))
	g.Expect(config.Common.Get("logger.count").MustInt()).Should(Equal(int64(9)))
	g.Expect(config.Proxy.Get("log-level").MustString()).Should(Equal("warn"))
	g.Expect(config.Proxy.Get("gc.batch-keys").MustInt()).Should(Equal(int64(501)))
}
