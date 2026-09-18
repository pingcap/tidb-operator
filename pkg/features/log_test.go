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

package features

import (
	"reflect"
	"testing"
)

func TestGeneratedFeatureLog(t *testing.T) {
	entries := FeatureLogs()
	if len(entries) == 0 {
		t.Fatal("empty log")
	}
	if len(hashToRev) != len(entries) {
		t.Fatal("hash index does not cover every revision")
	}
	seen := make(map[int]bool, len(entries))
	for hash, rev := range hashToRev {
		if rev < 0 || rev >= len(entries) || seen[rev] {
			t.Fatalf("invalid revision index %d", rev)
		}
		seen[rev] = true
		definitions, err := DefinitionsAt(hash)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(definitions, entries[rev]) {
			t.Fatalf("wrong snapshot for revision %d", rev)
		}
	}
	if rev, ok := hashToRev[CurrentFeatureGateDefinitionHash]; !ok || rev != len(entries)-1 {
		t.Fatal("current hash is stale")
	}
	prev := CurrentFeatureGateDefinitionHash
	definitions, err := DefinitionsAt(prev)
	if err != nil {
		t.Fatal(err)
	}
	name := definitions[0].Name
	definitions[0].Name = "Mutated"
	entries[len(entries)-1][0].Name = "Mutated"
	again, err := DefinitionsAt(prev)
	if err != nil {
		t.Fatal(err)
	}
	if again[0].Name != name {
		t.Fatal("caller mutated shared snapshot")
	}
	if _, err := DefinitionsAt("unknown"); err == nil {
		t.Fatal("unknown hash accepted")
	}
}
