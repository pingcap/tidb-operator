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
	"bytes"
	"encoding/json"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fixture = `package v1alpha1
 const (
 // +feature:unreloadable=
 // +feature:log=rev=0,stage=ALPHA,default=false
 // +feature:log=rev=1,stage=BETA,default=true
 A = "FeatureA"
 // +feature:unreloadable=
 // +feature:log=rev=0,stage=ALPHA,default=false
 // +feature:log=rev=1,stage=STABLE,default=true
 B = "FeatureB"
 )`

func logFrom(t *testing.T, source string) ([]feature, []entry) {
	t.Helper()
	features, err := parseFeatures([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := buildLog(features)
	if err != nil {
		t.Fatal(err)
	}
	return features, entries
}

func TestHashProtocol(t *testing.T) {
	_, entries := logFrom(t, fixture)
	wantJSON := []string{
		`{"prevHash":"","features":[{"name":"FeatureA","stage":"ALPHA","default":false},{"name":"FeatureB","stage":"ALPHA","default":false}]}`,
		`{"prevHash":"sha256:1a928b3db1a80122f70eaee946b46e68aabf1def5a3f8b94f31b26aa20108574","features":[{"name":"FeatureA","stage":"BETA","default":true},{"name":"FeatureB","stage":"STABLE","default":true}]}`,
	}
	wantHash := []string{
		"sha256:1a928b3db1a80122f70eaee946b46e68aabf1def5a3f8b94f31b26aa20108574",
		"sha256:9c52e93eae1c5e168caf04de0e01b82e97ba2ff07f1f6e79b9bb617b47817abc",
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries", len(entries))
	}
	for i, entry := range entries {
		data, err := json.Marshal(entry.Input)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != wantJSON[i] || entry.Hash != wantHash[i] {
			t.Fatalf("rev %d: %s, %s", i, data, entry.Hash)
		}
	}
}

func TestCanonicalOrder(t *testing.T) {
	features, original := logFrom(t, fixture)
	reordered := strings.ReplaceAll(fixture, "rev=0,stage=ALPHA,default=false", "default=false, stage=ALPHA, rev=0")
	otherFeatures, other := logFrom(t, reordered)
	if !reflect.DeepEqual(original, other) {
		t.Fatal("parameter ordering changed hashes")
	}
	features[0], features[1] = features[1], features[0]
	reversed, err := buildLog(features)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, reversed) {
		t.Fatal("declaration ordering changed hashes")
	}
	first, err := generate(otherFeatures, original, testComponents)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generate(otherFeatures, other, testComponents)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("generation is not deterministic")
	}
}

func TestValidation(t *testing.T) {
	cases := map[string]string{
		"missing history":     `A Feature = "FeatureA"`,
		"missing field":       "// +feature:log=rev=0,stage=ALPHA\nA Feature = \"FeatureA\"",
		"unknown field":       "// +feature:log=rev=0,stage=ALPHA,default=false,extra=x\nA Feature = \"FeatureA\"",
		"duplicate field":     "// +feature:log=rev=0,stage=ALPHA,default=false,rev=1\nA Feature = \"FeatureA\"",
		"invalid bool":        "// +feature:log=rev=0,stage=ALPHA,default=0\nA Feature = \"FeatureA\"",
		"invalid stage":       "// +feature:log=rev=0,stage=GA,default=true\nA Feature = \"FeatureA\"",
		"invalid default":     "// +feature:log=rev=0,stage=BETA,default=false\nA Feature = \"FeatureA\"",
		"gap":                 "// +feature:log=rev=1,stage=ALPHA,default=false\nA Feature = \"FeatureA\"",
		"duplicate rev":       "// +feature:log=rev=0,stage=ALPHA,default=false\n// +feature:log=rev=0,stage=BETA,default=true\nA Feature = \"FeatureA\"",
		"redundant":           "// +feature:log=rev=0,stage=ALPHA,default=false\n// +feature:log=rev=1,stage=ALPHA,default=false\nA Feature = \"FeatureA\"",
		"regression":          "// +feature:log=rev=0,stage=BETA,default=true\n// +feature:log=rev=1,stage=ALPHA,default=false\nA Feature = \"FeatureA\"",
		"deprecated terminal": "// +feature:log=rev=0,stage=DEPRECATED,default=false\n// +feature:log=rev=1,stage=DEPRECATED,default=true\nA Feature = \"FeatureA\"",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			features, err := parseFeatures([]byte("package v1alpha1\ntype Feature string\nconst (\n" + "// +feature:unreloadable=\n" + body + "\n)"))
			if err == nil {
				_, err = buildLog(features)
			}
			if err == nil {
				t.Fatal("expected invalid history to fail")
			}
		})
	}
}

func TestRebuildAndCheck(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"api/meta/v1alpha1", "pkg/features"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeTestComponents(t, root)
	source := filepath.Join(root, sourcePath)
	if err := os.WriteFile(source, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	if err := run(root, true); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "pkg", "features", "zz_generated.feature_log.go")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{path, filepath.Join(root, "api", "meta", "v1alpha1", "zz_generated.features.go"), filepath.Join(root, "pkg", "features", "zz_generated.reload.go")} {
		if err := os.Remove(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(root, true); err == nil {
		t.Fatal("missing generated file accepted")
	}
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("reconstruction changed output")
	}
	if err := os.WriteFile(path, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(root, true); err == nil {
		t.Fatal("stale output accepted")
	}
}

func TestGeneratedMetaDefinitions(t *testing.T) {
	source := strings.Replace(fixture, "// +feature:log=rev=0", "// FeatureA description.\n //\n // +feature:log=rev=0", 1)
	features, entries := logFrom(t, source)
	outputs, err := generate(features, entries, testComponents)
	if err != nil {
		t.Fatal(err)
	}
	if outputs[0].Path != "api/meta/v1alpha1/zz_generated.features.go" {
		t.Fatal("unexpected meta output")
	}
	generated := string(outputs[0].Data)
	for _, want := range []string{
		"// FeatureA description.",
		"+kubebuilder:validation:Enum=FeatureA;FeatureB",
		"A      Feature      = \"FeatureA\"",
		"AStage FeatureStage = \"BETA\"",
	} {
		if !strings.Contains(generated, want) {
			t.Fatalf("missing %q in generated meta definitions:\n%s", want, generated)
		}
	}
	generatedLog := string(outputs[1].Data)
	for _, name := range []string{"meta.A", "meta.B"} {
		if !strings.Contains(generatedLog, "Name: "+name+",") {
			t.Fatalf("generated log must reference feature constant %s:\n%s", name, generatedLog)
		}
	}
	for _, want := range []string{
		"var logs = [][]FeatureLog{",
		"var hashToRev = map[string]int{",
		"Stage: meta.FeatureStageAlpha",
		"Stage: meta.FeatureStageBeta",
		"Stage: meta.FeatureStageStable",
	} {
		if !strings.Contains(generatedLog, want) {
			t.Fatalf("generated log missing %q", want)
		}
	}
	if strings.Contains(generatedLog, "PrevHash") || strings.Contains(generatedLog, `Stage: "`) {
		t.Fatal("runtime log must use stage constants and omit previous hashes")
	}
	if strings.Contains(generatedLog, `Name: "`) {
		t.Fatal("generated log contains literal feature names")
	}
	if strings.Contains(generated, "+feature:log") {
		t.Fatal("history belongs in the source definitions only")
	}
	// Typed constants from the old source produce identical snapshots and hashes.
	legacy := strings.ReplaceAll(fixture, " = ", " Feature = ")
	_, oldEntries := logFrom(t, legacy)
	if !reflect.DeepEqual(entries, oldEntries) {
		t.Fatal("moving definitions changed hashes")
	}
}

func TestPrivateIotaDefinitions(t *testing.T) {
	const source = `package v1alpha1
 const (
 // +feature:unreloadable=
 // +feature:log=rev=0,stage=ALPHA,default=false
 // +feature:log=rev=1,stage=BETA,default=true
 _FeatureA = iota
 // +feature:unreloadable=
 // +feature:log=rev=0,stage=ALPHA,default=false
 // +feature:log=rev=1,stage=STABLE,default=true
 _FeatureB
 )
 const (
 _ = _FeatureA
 _ = _FeatureB
 )`
	features, entries := logFrom(t, source)
	_, oldEntries := logFrom(t, fixture)
	if !reflect.DeepEqual(entries, oldEntries) {
		t.Fatal("private iota definitions changed historical hashes")
	}
	if features[0].Identifier != "FeatureA" || features[1].Identifier != "FeatureB" {
		t.Fatal("private identifiers were not exported in generated definitions")
	}
	outputs, err := generate(features, entries, testComponents)
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range []string{"= _FeatureA", "= _FeatureB"} {
		if !strings.Contains(string(outputs[0].Data), reference) {
			t.Fatalf("missing generated reference %s", reference)
		}
	}
	// The numeric value assigned by iota is not a feature identity.
	reordered := `package v1alpha1
 const (
 // +feature:unreloadable=
 // +feature:log=rev=0,stage=ALPHA,default=false
 // +feature:log=rev=1,stage=STABLE,default=true
 _FeatureB = iota
 // +feature:unreloadable=
 // +feature:log=rev=0,stage=ALPHA,default=false
 // +feature:log=rev=1,stage=BETA,default=true
 _FeatureA
 )`
	reorderedFeatures, otherEntries := logFrom(t, reordered)
	if !reflect.DeepEqual(entries, otherEntries) {
		t.Fatal("iota ordering changed hashes")
	}
	reorderedOutputs, err := generate(reorderedFeatures, otherEntries, testComponents)
	if err != nil {
		t.Fatal(err)
	}
	generated := string(reorderedOutputs[0].Data)
	first := strings.Index(generated, "FeatureB      Feature")
	second := strings.Index(generated, "FeatureA      Feature")
	if first < 0 || second < 0 || first >= second {
		t.Fatalf("generated constants do not preserve source order:\n%s", generated)
	}
	if !strings.Contains(generated, "+kubebuilder:validation:Enum=FeatureB;FeatureA") {
		t.Fatal("enum does not preserve source order")
	}

	for _, invalid := range []string{
		strings.Replace(source, "_FeatureA = iota", "FeatureA = iota", 1),
		strings.Replace(source, "_FeatureA = iota", "_FeatureA = 0", 1),
		strings.Replace(source, "_FeatureA = iota", "_FeatureA", 1),
	} {
		if _, err := parseFeatures([]byte(invalid)); err == nil {
			t.Fatal("invalid iota definition accepted")
		}
	}
}

// Render bodies for focused generator tests; integration tests use gengo below.
func generate(features []feature, entries []entry, components map[string]string) ([]output, error) {
	outputs, err := generateBodies(features, entries, components)
	if err != nil {
		return nil, err
	}
	for i := range outputs {
		out := &outputs[i]
		source := "package " + out.Package + "\n"
		for _, imp := range out.Imports {
			source += "import " + imp + "\n"
		}
		out.Data, err = format.Source(append([]byte(source), out.Data...))
		if err != nil {
			return nil, err
		}
	}
	return outputs, nil
}

func run(root string, check bool) error {
	return execute(args{root: root, check: check, goHeaderFile: filepath.Join(root, "header.txt")}, []string{"./api/meta/v1alpha1"})
}
