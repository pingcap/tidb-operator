// Copyright 2026 PingCAP, Inc.
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

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testComponentSource = `package v1alpha1
 type Component string
 const (
 ComponentPD Component = "pd"
 ComponentTiKV Component = "tikv"
 ComponentDMWorker Component = "dm-worker"
 )`

var testComponents = map[string]string{"pd": "ComponentPD", "tikv": "ComponentTiKV", "dm-worker": "ComponentDMWorker"}

func writeTestComponents(t *testing.T, root string) {
	t.Helper()
	t.Chdir(root)
	for name, content := range map[string]string{"go.mod": "module example.com/features\n\ngo 1.25.0\n", "header.txt": "// Copyright YEAR PingCAP, Inc.\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, componentPath), []byte(testComponentSource), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateUnreloadable(t *testing.T) {
	components, err := parseComponents([]byte(testComponentSource))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(components, testComponents) {
		t.Fatal("wrong component catalog")
	}
	source := strings.Replace(fixture, "// +feature:unreloadable=", "// +feature:unreloadable=tikv,pd", 1)
	features, entries := logFrom(t, source)
	outputs, err := generate(features, entries, components)
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs[2].Data)
	if !strings.Contains(generated, "meta.A: {meta.ComponentPD, meta.ComponentTiKV}") ||
		!strings.Contains(generated, "meta.B: {}") {
		t.Fatalf("wrong generated map:\n%s", generated)
	}
	if strings.Contains(string(outputs[0].Data), "+feature:unreloadable") {
		t.Fatal("source marker leaked to generated API")
	}
	_, originalEntries := logFrom(t, fixture)
	if !reflect.DeepEqual(entries, originalEntries) {
		t.Fatal("reload metadata changed feature history hashes")
	}
	wildcard, wildcardEntries := logFrom(t, strings.Replace(source, "tikv,pd", "*", 1))
	components["future"] = "ComponentFuture"
	outputs, err = generate(wildcard, wildcardEntries, components)
	if err != nil {
		t.Fatal(err)
	}
	for _, identifier := range components {
		if !strings.Contains(string(outputs[2].Data), "meta."+identifier) {
			t.Fatalf("wildcard omitted %s", identifier)
		}
	}
}

func TestUnreloadableValidation(t *testing.T) {
	for _, value := range []string{"pd,pd", "*,pd", "pd,", ",pd"} {
		if _, err := parseUnreloadable(unreloadableMarker + value); err == nil {
			t.Fatalf("accepted invalid value %q", value)
		}
	}
	for _, value := range []string{"", "pd", "tikv,pd", "*"} {
		if _, err := parseUnreloadable(unreloadableMarker + value); err != nil {
			t.Fatal(err)
		}
	}
	source := strings.Replace(fixture, "// +feature:unreloadable=", "// +feature:unreloadable=typo", 1)
	features, entries := logFrom(t, source)
	if _, err := generate(features, entries, testComponents); err == nil {
		t.Fatal("unknown component accepted")
	}
	duplicate := strings.Replace(fixture, "// +feature:log=rev=0", "// +feature:unreloadable=\n // +feature:unreloadable=pd\n // +feature:log=rev=0", 1)
	if _, err := parseFeatures([]byte(duplicate)); err == nil {
		t.Fatal("duplicate marker accepted")
	}
	orphan := fixture + "\n// +feature:unreloadable=pd\nvar orphan int\n"
	if _, err := parseFeatures([]byte(orphan)); err == nil {
		t.Fatal("orphan marker accepted")
	}
}

func TestUnreloadableRequired(t *testing.T) {
	const source = `package v1alpha1
const (
 // +feature:unreloadable=
 // +feature:log=rev=0,stage=ALPHA,default=false
 _Example = iota
)`
	features, err := parseFeatures([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if features[0].Unreloadable == nil || len(features[0].Unreloadable) != 0 {
		t.Fatal("explicit empty marker must produce an empty component list")
	}
	missing := strings.Replace(source, " // +feature:unreloadable=\n", "", 1)
	if _, err := parseFeatures([]byte(missing)); err == nil || !strings.Contains(err.Error(), "Example: missing unreloadable marker") {
		t.Fatalf("expected missing unreloadable error, got %v", err)
	}
}
