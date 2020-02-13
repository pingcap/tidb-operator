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
	"bytes"
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

func TestTiKVConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	c := &TiKVConfig{
		ReadPool: &TiKVReadPoolConfig{
			Storage: &TiKVStorageReadPoolConfig{
				HighConcurrency: pointer.Int64Ptr(4),
			},
			Coprocessor: &TiKVCoprocessorReadPoolConfig{
				HighConcurrency: pointer.Int64Ptr(8),
			},
		},
		Storage: &TiKVStorageConfig{
			BlockCache: &TiKVBlockCacheConfig{
				Shared: pointer.BoolPtr(true),
			},
		},
	}
	jsonStr, err := json.Marshal(c)
	g.Expect(err).To(Succeed())
	g.Expect(jsonStr).NotTo(ContainSubstring("port"), "Expected empty fields to be omitted")
	var jsonUnmarshaled TiKVConfig
	err = json.Unmarshal(jsonStr, &jsonUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&jsonUnmarshaled).To(Equal(c))

	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err = encoder.Encode(c)
	g.Expect(err).To(Succeed())
	tStr := buff.String()
	g.Expect(tStr).To((Equal(`[storage]
  [storage.block-cache]
    shared = true

[readpool]
  [readpool.coprocessor]
    high-concurrency = 8
  [readpool.storage]
    high-concurrency = 4
`)))

	var tUnmarshaled TiKVConfig
	err = toml.Unmarshal([]byte(tStr), &tUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&tUnmarshaled).To(Equal(c))
}
