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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

const logPath = "pkg/features/zz_generated.feature_log.go"

type featureState struct {
	Stage   string
	Default bool
}

// Keys and stages retain the generated API constant names. No operator code
// from the current checkout or base commit is compiled or executed by lint.
type snapshot map[string]featureState

func parseLogs(source []byte) ([]snapshot, error) {
	file, err := parser.ParseFile(token.NewFileSet(), logPath, source, 0)
	if err != nil {
		return nil, err
	}
	var result []snapshot
	for _, decl := range file.Decls {
		vars, ok := decl.(*ast.GenDecl)
		if !ok || vars.Tok != token.VAR {
			continue
		}
		for _, spec := range vars.Specs {
			value := spec.(*ast.ValueSpec)
			for _, name := range value.Names {
				if name.Name != "logs" {
					continue
				}
				if result != nil {
					return nil, fmt.Errorf("duplicate logs declaration")
				}
				if len(value.Names) != 1 || len(value.Values) != 1 {
					return nil, fmt.Errorf("logs must have a single initializer")
				}
				result, err = parseSnapshots(value.Values[0])
				if err != nil {
					return nil, err
				}
			}
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("missing or empty generated logs")
	}
	return result, nil
}

func parseSnapshots(expr ast.Expr) ([]snapshot, error) {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("logs must be a [][]FeatureLog literal")
	}
	outer, ok := literal.Type.(*ast.ArrayType)
	if !ok || outer.Len != nil {
		return nil, fmt.Errorf("logs must be a slice of snapshots")
	}
	inner, ok := outer.Elt.(*ast.ArrayType)
	if !ok || inner.Len != nil {
		return nil, fmt.Errorf("logs must be a slice of snapshots")
	}
	typ, ok := inner.Elt.(*ast.Ident)
	if !ok || typ.Name != "FeatureLog" {
		return nil, fmt.Errorf("logs must contain FeatureLog entries")
	}
	result := make([]snapshot, 0, len(literal.Elts))
	for rev, expr := range literal.Elts {
		state, err := parseSnapshot(expr)
		if err != nil {
			return nil, fmt.Errorf("revision %d: %w", rev, err)
		}
		result = append(result, state)
	}
	return result, nil
}

func parseSnapshot(expr ast.Expr) (snapshot, error) {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok || len(literal.Elts) == 0 {
		return nil, fmt.Errorf("expected non-empty snapshot literal")
	}
	result := snapshot{}
	for _, expr := range literal.Elts {
		name, state, err := parseState(expr)
		if err != nil {
			return nil, err
		}
		if _, found := result[name]; found {
			return nil, fmt.Errorf("duplicate feature %s", name)
		}
		result[name] = state
	}
	return result, nil
}

func parseState(expr ast.Expr) (string, featureState, error) {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok {
		return "", featureState{}, fmt.Errorf("expected FeatureLog literal")
	}
	fields := map[string]ast.Expr{}
	for _, expr := range literal.Elts {
		field, ok := expr.(*ast.KeyValueExpr)
		if !ok {
			return "", featureState{}, fmt.Errorf("FeatureLog fields must be named")
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok {
			return "", featureState{}, fmt.Errorf("invalid FeatureLog field")
		}
		if key.Name != "Name" && key.Name != "Stage" && key.Name != "Default" {
			return "", featureState{}, fmt.Errorf("unknown FeatureLog field %s", key.Name)
		}
		if fields[key.Name] != nil {
			return "", featureState{}, fmt.Errorf("duplicate FeatureLog field %s", key.Name)
		}
		fields[key.Name] = field.Value
	}
	name, err := metaConstant(fields["Name"])
	if err != nil {
		return "", featureState{}, err
	}
	stage, err := metaConstant(fields["Stage"])
	if err != nil {
		return "", featureState{}, err
	}
	enabled, ok := fields["Default"].(*ast.Ident)
	if !ok || (enabled.Name != "true" && enabled.Name != "false") {
		return "", featureState{}, fmt.Errorf("default must be an explicit boolean literal")
	}
	return name, featureState{Stage: stage, Default: enabled.Name == "true"}, nil
}

func metaConstant(expr ast.Expr) (string, error) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", fmt.Errorf("name and stage must reference API constants")
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok || pkg.Name != "meta" {
		return "", fmt.Errorf("expected meta constant reference")
	}
	return selector.Sel.Name, nil
}
