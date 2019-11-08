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

package util

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"
)

func Checksum(src []byte) string {
	h := sha256.New()
	h.Write(src)
	out := h.Sum(nil)
	return hex.EncodeToString(out)
}

func DecodeCertPem(certByte []byte) (*x509.Certificate, error) {

	block, _ := pem.Decode(certByte)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// check if cert would be expired after refreshIntervalHour hours
func IsCertificateNeedRefresh(cert *x509.Certificate, refreshIntervalHour int) bool {
	now := time.Now()
	expireDate := cert.NotAfter
	internal := expireDate.Sub(now)
	return internal.Hours() <= float64(refreshIntervalHour)
}
