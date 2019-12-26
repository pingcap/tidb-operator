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
	"fmt"
	"hash/fnv"

	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"k8s.io/apimachinery/pkg/util/rand"
)

// HashContents hashes the contents using FNV hashing. The returned hash will be a safe encoded string to avoid bad words.
func HashContents(contents []byte) string {
	hf := fnv.New32()
	if len(contents) > 0 {
		hf.Write(contents)
	}
	return rand.SafeEncodeString(fmt.Sprint(hf.Sum32()))
}

// GetTidbPort return the tidb port
func (tac *TiDBAccessConfig) GetTidbPort() int32 {
	if tac.Port != 0 {
		return tac.Port
	}
	return constants.DefaultTidbPort
}

// GetTidbUser return the tidb user
func (tac *TiDBAccessConfig) GetTidbUser() string {
	if len(tac.User) != 0 {
		return tac.User
	}
	return constants.DefaultTidbUser
}

// GetTidbEndpoint return the tidb endpoint for access tidb cluster directly
func (tac *TiDBAccessConfig) GetTidbEndpoint() string {
	return fmt.Sprintf("%s_%d", tac.Host, tac.GetTidbPort())
}
