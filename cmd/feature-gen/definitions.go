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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const sourcePath = "api/meta/v1alpha1/feature.go"
const componentPath = "api/meta/v1alpha1/types.go"

func prepare(root string) ([]output, error) {
	source, err := os.ReadFile(filepath.Join(root, sourcePath))
	if err != nil {
		return nil, err
	}
	features, err := parseFeatures(source)
	if err != nil {
		return nil, err
	}
	entries, err := buildLog(features)
	if err != nil {
		return nil, err
	}
	componentSource, err := os.ReadFile(filepath.Join(root, componentPath))
	if err != nil {
		return nil, err
	}
	components, err := parseComponents(componentSource)
	if err != nil {
		return nil, err
	}
	outputs, err := generateBodies(features, entries, components)
	if err != nil {
		return nil, err
	}
	return outputs, nil
}

type output struct {
	Path    string
	Data    []byte
	Package string
	Imports []string
}

func generateBodies(features []feature, entries []entry, components map[string]string) ([]output, error) {
	var stages, log bytes.Buffer
	stages.WriteString("// Feature defines a supported feature of a TiDB cluster.\n//\n")
	names := make([]string, 0, len(features))
	identifiers := make(map[string]string, len(features))
	for _, f := range features {
		names = append(names, f.History[0].Definition.Name)
		identifiers[f.History[0].Definition.Name] = f.Identifier
	}
	fmt.Fprintf(&stages, "// +kubebuilder:validation:Enum=%s\n// +enum\ntype Feature string\n\nconst (\n", strings.Join(names, ";"))
	for _, f := range features {
		latest := f.History[len(f.History)-1].Definition
		comments := slices.Clone(f.Comments)
		for len(comments) > 0 && strings.TrimSpace(comments[len(comments)-1]) == "//" {
			comments = comments[:len(comments)-1]
		}
		for _, comment := range comments {
			fmt.Fprintln(&stages, comment)
		}
		fmt.Fprintf(&stages, "%s Feature = %q\n", f.Identifier, latest.Name)
		fmt.Fprintf(&stages, "%sStage FeatureStage = %q\n", f.Identifier, latest.Stage)
		fmt.Fprintf(&stages, "_ = %s\n\n", f.SourceIdentifier)
	}
	stages.WriteString(")\n")
	fmt.Fprintf(&log, "const CurrentFeatureGateDefinitionHash = %q\n\n", entries[len(entries)-1].Hash)
	stageNames := map[string]string{
		stageAlpha:      "FeatureStageAlpha",
		stageBeta:       "FeatureStageBeta",
		stageStable:     "FeatureStageStable",
		stageDeprecated: "FeatureStageDeprecated",
	}
	log.WriteString("var logs = [][]FeatureLog{\n")
	for _, e := range entries {
		log.WriteString("{\n")
		for _, d := range e.Input.Features {
			fmt.Fprintf(&log, "{Name: meta.%s, Stage: meta.%s, Default: %t},\n",
				identifiers[d.Name], stageNames[d.Stage], d.Default)
		}
		log.WriteString("},\n")
	}
	log.WriteString("}\n\nvar hashToRev = map[string]int{\n")
	for rev, e := range entries {
		fmt.Fprintf(&log, "%q: %d,\n", e.Hash, rev)
	}
	log.WriteString("}\n")
	reload, err := generateUnreloadable(features, components, "")
	if err != nil {
		return nil, err
	}
	return []output{
		{Path: "api/meta/v1alpha1/zz_generated.features.go", Package: "v1alpha1", Data: stages.Bytes()},
		{Path: "pkg/features/zz_generated.feature_log.go", Package: runtimePackageName, Data: log.Bytes(), Imports: []string{metaImport}},
		{Path: "pkg/features/zz_generated.reload.go", Package: runtimePackageName, Data: reload, Imports: []string{metaImport}},
	}, nil
}
