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

package v1alpha1

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
)

func TestTiDBConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	procs := uint(8)
	probability := 0.8
	c := &TiDBConfig{
		Performance: &Performance{
			MaxProcs:            &procs,
			CrossJoin:           pointer.BoolPtr(true),
			FeedbackProbability: &probability,
		},
	}
	jsonStr, err := json.Marshal(c)
	g.Expect(err).To(Succeed())
	g.Expect(jsonStr).To(ContainSubstring("cross-join"))
	g.Expect(jsonStr).To(ContainSubstring("0.8"))
	g.Expect(jsonStr).NotTo(ContainSubstring("port"), "Expected empty fields to be omitted")
	var jsonUnmarshaled TiDBConfig
	err = json.Unmarshal(jsonStr, &jsonUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&jsonUnmarshaled).To(Equal(c))

	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err = encoder.Encode(c)
	g.Expect(err).To(Succeed())
	tStr := buff.String()
	g.Expect(tStr).To((Equal(`[performance]
  max-procs = 8
  feedback-probability = 0.8
  cross-join = true
`)))

	var tUnmarshaled TiDBConfig
	err = toml.Unmarshal([]byte(tStr), &tUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&tUnmarshaled).To(Equal(c))
}
