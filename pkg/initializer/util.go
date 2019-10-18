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

package initializer

import (
	"crypto/sha256"
	"encoding/hex"
	certUtil "github.com/pingcap/tidb-operator/pkg/util"
	core "k8s.io/api/core/v1"
)

func Checksum(src []byte) string {
	h := sha256.New()
	h.Write(src)
	out := h.Sum(nil)
	return hex.EncodeToString(out)
}

// generate Secret to create a new CA cert
func (initializer *Initializer) GenerateNewSecretForInitializer(namespace string) (*core.Secret, error) {
	pubKey, pKey, err := certUtil.GenerateRSAKeyPair(namespace, admissionWebhookServiceName)
	if err != nil {
		return nil, err
	}
	secret := core.Secret{}
	secret.Name = SecretName
	secret.Data = map[string][]byte{
		"cert.pem": pubKey,
		"key.pem":  pKey,
	}
	return &secret, nil
}
