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

// Package random is copied from
// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
package random

import (
	"math/rand"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz1234567890"

const (
	// 6 bits to represent a letter index.
	letterIdxBits = 6
	// All 1-bits, as many as letterIdxBits.
	letterIdxMask = 1<<letterIdxBits - 1
	// # of letter indices fitting in 63 bits.
	letterIdxMax = 63 / letterIdxBits
)

// Random returns a random string with fixed length.
func Random(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	//nolint:gosec // no need to use cryptographically secure random number
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			//nolint:gosec // no need to use cryptographically secure random number
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			if err := sb.WriteByte(letterBytes[idx]); err != nil {
				panic(err)
			}
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}
