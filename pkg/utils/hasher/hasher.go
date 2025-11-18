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

package hasher

import (
	"fmt"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/util/rand"

	hashutil "github.com/pingcap/tidb-operator/v2/third_party/kubernetes/pkg/util/hash"
)

// Hash an obj by this struct
// NOTE: It may be changed if struct's fields are added or deleted. e.g. dep is upgraded
func Hash(obj any) string {
	hasher := fnv.New32a()
	hashutil.DeepHashObject(hasher, obj)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}
