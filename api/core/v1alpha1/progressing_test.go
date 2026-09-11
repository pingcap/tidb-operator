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

package v1alpha1

import (
	"encoding/json"
	"testing"

	"k8s.io/utils/ptr"
)

func TestProgressingRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name        string
		progressing *bool
	}{
		{name: "omitted"},
		{name: "paused", progressing: ptr.To(false)},
		{name: "enabled", progressing: ptr.To(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			group := &TiKVGroup{Spec: TiKVGroupSpec{Progressing: tc.progressing}}
			data, err := json.Marshal(group)
			if err != nil {
				t.Fatal(err)
			}
			var decoded TiKVGroup
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if !ptr.Equal(tc.progressing, decoded.Spec.Progressing) {
				t.Fatalf("progressing changed during JSON round trip: %s", data)
			}
			copied := group.DeepCopy()
			if !ptr.Equal(tc.progressing, copied.Spec.Progressing) {
				t.Fatal("progressing changed during deep copy")
			}
			if copied.Spec.Progressing != nil {
				*copied.Spec.Progressing = !*copied.Spec.Progressing
				if *copied.Spec.Progressing == *group.Spec.Progressing {
					t.Fatal("deep copy shares progressing pointer with original")
				}
			}
		})
	}
}
