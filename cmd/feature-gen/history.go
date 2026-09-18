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

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const unreloadableMarker = "+feature:unreloadable="
const marker = "+feature:log="

const (
	stageAlpha      = "ALPHA"
	stageBeta       = "BETA"
	stageStable     = "STABLE"
	stageDeprecated = "DEPRECATED"
)

var (
	featureNamePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	revisionPattern    = regexp.MustCompile(`^(0|[1-9]\d*)$`)
)

// Definition describes a feature at a revision. Field order and JSON tags are
// part of the immutable hash protocol.
type definition struct {
	Name    string `json:"name"`
	Stage   string `json:"stage"`
	Default bool   `json:"default"`
}

// HashInput is the canonical hash payload; its field order is immutable.
type hashInput struct {
	PrevHash string       `json:"prevHash"`
	Features []definition `json:"features"`
}

// Change records a definition at its revision index.
type change struct {
	Rev        uint64
	Definition definition
}

// Feature holds a source declaration and its history.
type feature struct {
	SourceIdentifier string
	Identifier       string
	Comments         []string
	History          []change
	Unreloadable     []string
}

// Entry holds a complete snapshot and its chained hash.
type entry struct {
	Hash  string
	Input hashInput
}

// parseFeatures uses the source AST to preserve declaration order and reject
// misplaced markers.
func parseFeatures(source []byte) ([]feature, error) {
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var result []feature
	names := map[string]bool{}
	consumed := map[*ast.Comment]bool{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		var previousValues []ast.Expr
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			if len(value.Values) > 0 {
				previousValues = value.Values
			} else {
				inherited := *value
				inherited.Values = previousValues
				value = &inherited
			}
			if !isFeatureDefinition(value) {
				continue
			}
			doc := value.Doc
			if doc == nil && !gen.Lparen.IsValid() {
				doc = gen.Doc
			}
			f, err := parseFeature(value, doc, consumed)
			if err != nil {
				return nil, err
			}
			name := f.History[0].Definition.Name
			if names[name] {
				return nil, fmt.Errorf("duplicate feature name %q", name)
			}
			names[name] = true
			result = append(result, f)
		}
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			isMarker := strings.Contains(comment.Text, "+feature:log") ||
				strings.Contains(comment.Text, "+feature:unreloadable")
			if isMarker && !consumed[comment] {
				return nil, fmt.Errorf("feature marker must be a doc comment on a feature constant: %s", comment.Text)
			}
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no feature definitions found")
	}
	return result, nil
}

func isFeatureDefinition(value *ast.ValueSpec) bool {
	if len(value.Names) == 1 && value.Names[0].Name == "_" {
		return false
	}
	if value.Type == nil {
		return true
	}
	// Accept the old typed constants when checking pre-migration history.
	typ, ok := value.Type.(*ast.Ident)
	return ok && typ.Name == "Feature"
}

// featureIdentity accepts private iota definitions and legacy string definitions
// so append-only verification also works across the source-format migration.
func featureIdentity(value *ast.ValueSpec) (featureName, constantName string, err error) {
	if len(value.Names) != 1 || len(value.Values) != 1 {
		return "", "", fmt.Errorf("each feature must have one name and an iota initializer")
	}
	identifier := value.Names[0].Name
	if init, ok := value.Values[0].(*ast.Ident); ok && init.Name == "iota" {
		if !strings.HasPrefix(identifier, "_") {
			return "", "", fmt.Errorf("%s: iota feature constants must start with _", identifier)
		}
		name := strings.TrimPrefix(identifier, "_")
		if !featureNamePattern.MatchString(name) {
			return "", "", fmt.Errorf("invalid feature name %q", name)
		}
		return name, name, nil
	}
	literal, ok := value.Values[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", "", fmt.Errorf("%s: expected iota initializer", identifier)
	}
	name, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", "", err
	}
	if !featureNamePattern.MatchString(name) {
		return "", "", fmt.Errorf("invalid feature name %q", name)
	}
	return name, identifier, nil
}

func parseFeature(value *ast.ValueSpec, doc *ast.CommentGroup, consumed map[*ast.Comment]bool) (feature, error) {
	name, identifier, err := featureIdentity(value)
	if err != nil {
		return feature{}, err
	}
	f := feature{Identifier: identifier, SourceIdentifier: value.Names[0].Name}
	if doc != nil {
		for _, comment := range doc.List {
			line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(line, "+feature:unreloadable") {
				if f.Unreloadable != nil {
					return feature{}, fmt.Errorf("%s: duplicate unreloadable marker", name)
				}
				f.Unreloadable, err = parseUnreloadable(line)
				if err != nil {
					return feature{}, fmt.Errorf("%s: %w", name, err)
				}
				consumed[comment] = true
				continue
			}
			if !strings.HasPrefix(line, "+feature:log") {
				f.Comments = append(f.Comments, comment.Text)
				continue
			}
			consumed[comment] = true
			c, err := parseChange(name, line)
			if err != nil {
				return feature{}, fmt.Errorf("%s: %w", name, err)
			}
			if len(f.History) > 0 {
				previous := f.History[len(f.History)-1]
				if c.Rev <= previous.Rev {
					return feature{}, fmt.Errorf("%s: revisions must increase", name)
				}
				if err := validateTransition(previous.Definition, c.Definition); err != nil {
					return feature{}, err
				}
			}
			f.History = append(f.History, c)
		}
	}
	if len(f.History) == 0 {
		return feature{}, fmt.Errorf("%s: missing feature log", name)
	}
	if f.Unreloadable == nil {
		return feature{}, fmt.Errorf("%s: missing unreloadable marker; declare +feature:unreloadable= explicitly, even when empty", name)
	}
	return f, nil
}

func parseChange(name, line string) (change, error) {
	var c change
	if !strings.HasPrefix(line, marker) {
		return c, fmt.Errorf("invalid marker %q", line)
	}
	fields := map[string]string{}
	for _, field := range strings.Split(strings.TrimPrefix(line, marker), ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
		if !ok {
			return c, fmt.Errorf("invalid parameter %q", field)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key != "rev" && key != "stage" && key != "default" {
			return c, fmt.Errorf("unknown parameter %q", key)
		}
		if _, exists := fields[key]; exists {
			return c, fmt.Errorf("duplicate parameter %q", key)
		}
		fields[key] = value
	}
	if len(fields) != 3 {
		return c, fmt.Errorf("rev, stage and default are required")
	}
	if !revisionPattern.MatchString(fields["rev"]) {
		return c, fmt.Errorf("invalid revision %q", fields["rev"])
	}
	rev, err := strconv.ParseUint(fields["rev"], 10, 64)
	if err != nil {
		return c, err
	}
	if fields["default"] != "true" && fields["default"] != "false" {
		return c, fmt.Errorf("default must be true or false")
	}
	d := definition{Name: name, Stage: fields["stage"], Default: fields["default"] == "true"}
	switch d.Stage {
	case stageAlpha:
		if d.Default {
			return c, fmt.Errorf("ALPHA default must be false")
		}
	case stageBeta, stageStable:
		if !d.Default {
			return c, fmt.Errorf("%s default must be true", d.Stage)
		}
	case stageDeprecated:
	default:
		return c, fmt.Errorf("unknown stage %q", d.Stage)
	}
	return change{Rev: rev, Definition: d}, nil
}

func validateTransition(old, next definition) error {
	if old == next {
		return fmt.Errorf("%s: redundant feature log", next.Name)
	}
	order := map[string]int{stageAlpha: 0, stageBeta: 1, stageStable: 2, stageDeprecated: 3}
	if order[next.Stage] <= order[old.Stage] {
		return fmt.Errorf("%s: invalid stage transition %s -> %s", next.Name, old.Stage, next.Stage)
	}
	return nil
}

// buildLog reconstructs complete snapshots and canonical hashes in revision order.
func buildLog(features []feature) ([]entry, error) {
	revisions := map[uint64][]definition{}
	for _, f := range features {
		for _, c := range f.History {
			revisions[c.Rev] = append(revisions[c.Rev], c.Definition)
		}
	}
	state := map[string]definition{}
	entries := make([]entry, 0, len(revisions))
	prev := ""
	for rev := range uint64(len(revisions)) {
		changes, ok := revisions[rev]
		if !ok {
			return nil, fmt.Errorf("missing revision %d; revisions must start at 0 and be contiguous", rev)
		}
		for _, d := range changes {
			state[d.Name] = d
		}
		snapshot := make([]definition, 0, len(state))
		for _, d := range state {
			snapshot = append(snapshot, d)
		}
		sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].Name < snapshot[j].Name })
		input := hashInput{PrevHash: prev, Features: snapshot}
		data, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		hash := "sha256:" + hex.EncodeToString(sum[:])
		entries = append(entries, entry{Hash: hash, Input: input})
		prev = hash
	}
	return entries, nil
}

// parseUnreloadable validates and parses restart metadata.
func parseUnreloadable(line string) ([]string, error) {
	if !strings.HasPrefix(line, unreloadableMarker) {
		return nil, fmt.Errorf("invalid unreloadable marker %q", line)
	}
	value := strings.TrimSpace(strings.TrimPrefix(line, unreloadableMarker))
	if value == "" {
		return []string{}, nil
	}
	components := strings.Split(value, ",")
	seen := make(map[string]bool)
	for i, component := range components {
		component = strings.TrimSpace(component)
		if component == "" || seen[component] {
			return nil, fmt.Errorf("empty or duplicate unreloadable component %q", component)
		}
		if component == "*" && len(components) != 1 {
			return nil, fmt.Errorf("wildcard must be used alone")
		}
		seen[component] = true
		components[i] = component
	}
	return components, nil
}
