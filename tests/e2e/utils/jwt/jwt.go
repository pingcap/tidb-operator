// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jwt

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

// ref: https://raw.githubusercontent.com/CbcWestwolf/generate_jwt/master/JWKS.json
//
//nolint:lll // this is a long string
const jwksJSON = `
{
  "keys": [
    {
      "alg": "RS256",
      "e": "AQAB",
      "kid": "the-key-id-0",
      "kty": "RSA",
      "n": "q8G5n9XBidxmBMVJKLOBsmdOHrCqGf17y9-VUXingwDUZxRp2XbuLZLbJtLgcln1lC0L9BsogrWf7-pDhAzWovO6Ai4Aybu00tJ2u0g4j1aLiDdsy0gyvSb5FBoL08jFIH7t_JzMt4JpF487AjzvITwZZcnsrB9a9sdn2E5B_aZmpDGi2-Isf5osnlw0zvveTwiMo9ba416VIzjntAVEvqMFHK7vyHqXbfqUPAyhjLO-iee99Tg5AlGfjo1s6FjeML4xX7sAMGEy8FVBWNfpRU7ryTWoSn2adzyA_FVmtBvJNQBCMrrAhXDTMJ5FNi8zHhvzyBKHU0kBTS1UNUbP9w",
      "use": "sig"
    }
  ]
}
`

func GenerateJWKSSecret(ns, name string) v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Data: map[string][]byte{
			v1alpha1.TiDBAuthTokenJWKS: []byte(jwksJSON),
		},
	}
}

func GenerateJWT(kid, sub, email, iss string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot get the runtime info")
	}
	curDir := filepath.Dir(filename)
	toolPath := filepath.Join(curDir, "..", "..", "..", "..", "_output", "bin", "generate_jwt")
	absPath, err := filepath.Abs(toolPath)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(absPath, "--kid", kid, "--sub", sub, "--email", email, "--iss", iss)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// the output contains both the jwt and the public key, like:
	// -----BEGIN PUBLIC KEY-----
	// MIIBCgKCAQEAq8G5n9XBidxmBMVJKLOBsmdOHrCqGf17y9+VUXingwDUZxRp2Xbu
	// LZLbJtLgcln1lC0L9BsogrWf7+pDhAzWovO6Ai4Aybu00tJ2u0g4j1aLiDdsy0gy
	// vSb5FBoL08jFIH7t/JzMt4JpF487AjzvITwZZcnsrB9a9sdn2E5B/aZmpDGi2+Is
	// f5osnlw0zvveTwiMo9ba416VIzjntAVEvqMFHK7vyHqXbfqUPAyhjLO+iee99Tg5
	// AlGfjo1s6FjeML4xX7sAMGEy8FVBWNfpRU7ryTWoSn2adzyA/FVmtBvJNQBCMrrA
	// hXDTMJ5FNi8zHhvzyBKHU0kBTS1UNUbP9wIDAQAB
	// -----END PUBLIC KEY-----

	//nolint:lll // this is a long string
	// eyJhbGciOiJSUzI1NiIsImtpZCI6InRoZS1rZXktaWQtMCIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InVzZXJAcGluZ2NhcC5jb20iLCJleHAiOjE3MjczNDE3NzEsImlhdCI6MTcyNzM0MDg3MSwiaXNzIjoiaXNzdWVyLWFiYyIsInN1YiI6InVzZXJAcGluZ2NhcC5jb20ifQ.jYdvNNvVJ4NmbMuG4ilU0_ajBwnB7P17zaJYwEQDtLrG6N-uWp75-dCX48CFX0OSldahI9-bpM_tzaCYeuzjMF5Ee2sMq1Mbz7FHg15oW8yYCfOeTnFG8f0YH15Ql3p62TJ-G7IVk9rDKIJnHFpeNSvbue_V0tiH5ZCaJztKDidJ6HkA6zoW90mltojVtaVV05d-amLbYIWB5nRX80mJpAOB_WBUK8e_kyPzL11-j9-34TFvVx78Bd4O43Jx-eC7VZUKNtFCewM47sLZbgabFNY-_f3bv-Nu_LaWXlaWq6BYgNErr7vIEJjjyeTe6vuDE6ziyB_QM-cwiNK8gQ7sAA

	// we only need the jwt part
	slices := strings.Split(string(out), "\n")
	jwt := slices[len(slices)-2]
	return jwt, nil
}
