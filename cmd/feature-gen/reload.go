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
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"sort"
	"strconv"
)

// parseComponents reads the API's component constants so wildcard expansion
// automatically includes new components without a second hand-maintained list.
func parseComponents(source []byte) (map[string]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), componentPath, source, 0)
	if err != nil {
		return nil, err
	}
	components := make(map[string]string)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			typ, ok := value.Type.(*ast.Ident)
			if !ok || typ.Name != "Component" {
				continue
			}
			if len(value.Names) != 1 || len(value.Values) != 1 {
				return nil, fmt.Errorf("each Component must have one explicit string value")
			}
			literal, ok := value.Values[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return nil, fmt.Errorf("component must be a string literal")
			}
			name, err := strconv.Unquote(literal.Value)
			if err != nil {
				return nil, err
			}
			if name == "" || components[name] != "" {
				return nil, fmt.Errorf("empty or duplicate component %q", name)
			}
			components[name] = value.Names[0].Name
		}
	}
	if len(components) == 0 {
		return nil, fmt.Errorf("no Component definitions found")
	}
	return components, nil
}

func generateUnreloadable(features []feature, components map[string]string, header string) ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(header)
	out.WriteString("// unreloadable lists components that require a restart when a feature changes.\n")
	out.WriteString("var unreloadable = map[meta.Feature][]meta.Component{\n")
	for _, f := range features {
		names := slices.Clone(f.Unreloadable)
		if len(names) == 1 && names[0] == "*" {
			names = make([]string, 0, len(components))
			for name := range components {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		fmt.Fprintf(&out, "meta.%s: {", f.Identifier)
		for _, name := range names {
			identifier, ok := components[name]
			if !ok {
				return nil, fmt.Errorf("%s: unknown unreloadable component %q", f.Identifier, name)
			}
			fmt.Fprintf(&out, "meta.%s,", identifier)
		}
		out.WriteString("},\n")
	}
	out.WriteString("}\n")
	return out.Bytes(), nil
}
