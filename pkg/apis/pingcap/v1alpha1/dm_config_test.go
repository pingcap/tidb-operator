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

func TestDMMasterConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	c := &MasterConfig{
		RPCTimeoutStr: pointer.StringPtr("40s"),
		RPCRateLimit:  pointer.Float64Ptr(15),
		DMSecurityConfig: DMSecurityConfig{
			SSLCA:   pointer.StringPtr("/var/lib/dm-master-tls/ca.crt"),
			SSLCert: pointer.StringPtr("/var/lib/dm-master-tls/tls.crt"),
			SSLKey:  pointer.StringPtr("/var/lib/dm-master-tls/tls.key"),
		},
	}
	jsonStr, err := json.Marshal(c)
	g.Expect(err).To(Succeed())
	g.Expect(jsonStr).To(ContainSubstring("rpc-rate-limit"))
	g.Expect(jsonStr).To(ContainSubstring("40s"))
	g.Expect(jsonStr).NotTo(ContainSubstring("rpc-rate-burst"), "Expected empty fields to be omitted")
	var jsonUnmarshaled MasterConfig
	err = json.Unmarshal(jsonStr, &jsonUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&jsonUnmarshaled).To(Equal(c))

	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err = encoder.Encode(c)
	g.Expect(err).To(Succeed())
	tStr := buff.String()
	g.Expect(tStr).To((Equal(`rpc-timeout = "40s"
rpc-rate-limit = 15.0
ssl-ca = "/var/lib/dm-master-tls/ca.crt"
ssl-cert = "/var/lib/dm-master-tls/tls.crt"
ssl-key = "/var/lib/dm-master-tls/tls.key"
`)))

	var tUnmarshaled MasterConfig
	err = toml.Unmarshal([]byte(tStr), &tUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&tUnmarshaled).To(Equal(c))
}

func TestDMWorkerConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	c := &WorkerConfig{
		KeepAliveTTL: pointer.Int64Ptr(15),
		DMSecurityConfig: DMSecurityConfig{
			SSLCA:   pointer.StringPtr("/var/lib/dm-worker-tls/ca.crt"),
			SSLCert: pointer.StringPtr("/var/lib/dm-worker-tls/tls.crt"),
			SSLKey:  pointer.StringPtr("/var/lib/dm-worker-tls/tls.key"),
		},
	}
	jsonStr, err := json.Marshal(c)
	g.Expect(err).To(Succeed())
	g.Expect(jsonStr).NotTo(ContainSubstring("log-file"), "Expected empty fields to be omitted")
	var jsonUnmarshaled WorkerConfig
	err = json.Unmarshal(jsonStr, &jsonUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&jsonUnmarshaled).To(Equal(c))

	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err = encoder.Encode(c)
	g.Expect(err).To(Succeed())
	tStr := buff.String()
	g.Expect(tStr).To((Equal(`keepalive-ttl = 15
ssl-ca = "/var/lib/dm-worker-tls/ca.crt"
ssl-cert = "/var/lib/dm-worker-tls/tls.crt"
ssl-key = "/var/lib/dm-worker-tls/tls.key"
`)))

	var tUnmarshaled WorkerConfig
	err = toml.Unmarshal([]byte(tStr), &tUnmarshaled)
	g.Expect(err).To(Succeed())
	g.Expect(&tUnmarshaled).To(Equal(c))
}
