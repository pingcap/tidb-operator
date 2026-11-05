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

package util

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredentialArgumentsAreNotLogged(t *testing.T) {
	for _, file := range []string{"backup/backup.go", "backup/manager.go", "restore/restore.go", "restore/manager.go"} {
		source, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", file), nil, 0)
		require.NoError(t, err)
		ast.Inspect(source, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || (pkg.Name != "klog" && (pkg.Name != "fmt" || selector.Sel.Name != "Errorf")) {
				return true
			}
			for _, arg := range call.Args {
				ast.Inspect(arg, func(n ast.Node) bool {
					if id, ok := n.(*ast.Ident); ok {
						require.NotContains(t, []string{"fullArgs", "crData"}, id.Name, "credential logging in %s", file)
					}
					return true
				})
			}
			return true
		})
	}
}
