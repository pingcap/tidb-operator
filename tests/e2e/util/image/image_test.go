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

package image

import (
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestReadImagesFromValues(t *testing.T) {
	tests := []struct {
		name       string
		values     string
		keys       sets.String
		wantImages []string
	}{
		{
			name: "basic",
			values: `
image: pingcap/tidb:v3.0.4
foo:
  image: busybox:latest
  mysql:
    image: pingcap/tidb-monitor-reloader:v1.0.1
`,
			keys: nil,
			wantImages: []string{
				"pingcap/tidb-monitor-reloader:v1.0.1",
				"pingcap/tidb:v3.0.4",
				"busybox:latest",
			},
		},
		{
			name: "basic",
			values: `
image: pingcap/tidb:v3.0.4
foo:
  image: busybox:latest
  mysql:
    image: pingcap/tidb-monitor-reloader:v1.0.1
`,
			keys: sets.NewString(".image", ".foo.image"),
			wantImages: []string{
				"pingcap/tidb:v3.0.4",
				"busybox:latest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := ioutil.TempFile("", "values")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name()) // clean up
			_, err = tmpfile.Write([]byte(tt.values))
			if err != nil {
				t.Fatal(err)
			}
			got, err := readImagesFromValues(tmpfile.Name(), tt.keys)
			if err != nil {
				t.Error(err)
			}
			sort.Strings(got)
			sort.Strings(tt.wantImages)
			if diff := cmp.Diff(tt.wantImages, got); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}
