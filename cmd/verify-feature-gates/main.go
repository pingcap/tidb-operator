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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// validateFeatureName checks if a feature name follows Go naming conventions
func validateFeatureName(name string) bool {
	// Feature names must start with uppercase letter and contain only alphanumeric characters
	matched, _ := regexp.MatchString(`^[A-Z][a-zA-Z0-9]*$`, name)
	return matched
}

const (
	featureFile = "api/meta/v1alpha1/feature.go"
	reloadFile  = "pkg/features/reload.go"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-h" {
		fmt.Println("Feature Gates Consistency Verifier")
		fmt.Println("This tool verifies consistency between:")
		fmt.Println("- Feature constants in", featureFile)
		fmt.Println("- Unreloadable map entries in", reloadFile)
		fmt.Println("- Kubebuilder Enum annotations")
		os.Exit(0)
	}

	// Find root directory
	root, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	featurePath := filepath.Join(root, featureFile)
	reloadPath := filepath.Join(root, reloadFile)

	// Verify files exist
	if err := checkFilesExist(featurePath, reloadPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Checking feature gates consistency...")

	// Extract feature definitions
	features, err := extractFeatureConstants(featurePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting feature constants: %v\n", err)
		os.Exit(1)
	}

	// Extract unreloadable map keys
	unreloadableKeys, err := extractUnreloadableKeys(reloadPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting unreloadable keys: %v\n", err)
		os.Exit(1)
	}

	// Extract enum annotation features and validate them
	enumFeatures, err := extractEnumAnnotation(featurePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting enum annotation: %v\n", err)
		os.Exit(1)
	}

	// Validate all extracted feature names
	for _, feature := range enumFeatures {
		if !validateFeatureName(feature) {
			fmt.Fprintf(os.Stderr, "Error: Invalid feature name in enum annotation: '%s'\n", feature)
			os.Exit(1)
		}
	}

	// Display findings
	displayResults(features, unreloadableKeys, enumFeatures)

	// Validate consistency
	exitCode := validateConsistency(features, unreloadableKeys, enumFeatures)
	if exitCode == 0 {
		fmt.Println("✅ Feature gates consistency check passed!")
	}
	os.Exit(exitCode)
}

// findProjectRoot finds the project root directory by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Limit traversal depth to prevent infinite loops
	maxDepth := 10
	currentDepth := 0

	for currentDepth < maxDepth {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		currentDepth++
	}
	return "", fmt.Errorf("could not find project root (no go.mod found within %d directories)", maxDepth)
}

// checkFilesExist verifies that required files exist and are readable
func checkFilesExist(files ...string) error {
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("file %s not found or not readable: %w", filepath.Base(file), err)
		}
	}
	return nil
}

// extractFeatureConstants extracts Feature constant definitions from feature.go
func extractFeatureConstants(filename string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	var features []string

	// Look for const declarations
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Check if this is a Feature type constant
			if valueSpec.Type != nil {
				if ident, ok := valueSpec.Type.(*ast.Ident); ok && ident.Name == "Feature" {
					for _, name := range valueSpec.Names {
						if !validateFeatureName(name.Name) {
							return nil, fmt.Errorf("invalid feature name '%s': must start with uppercase letter and contain only alphanumeric characters",
								name.Name)
						}
						features = append(features, name.Name)
					}
				}
			}
		}
	}

	sort.Strings(features)
	return features, nil
}

// extractUnreloadableKeys extracts keys from the unreloadable map in reload.go
func extractUnreloadableKeys(filename string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	var keys []string

	// Look for variable declarations
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Look for the unreloadable variable
			for i, name := range valueSpec.Names {
				if name.Name == "unreloadable" && i < len(valueSpec.Values) {
					if compositeLit, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
						keys = extractMapKeys(compositeLit)
					}
				}
			}
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// extractMapKeys extracts keys from a map literal AST node
func extractMapKeys(compositeLit *ast.CompositeLit) []string {
	var keys []string

	for _, elt := range compositeLit.Elts {
		if keyValue, ok := elt.(*ast.KeyValueExpr); ok {
			if selectorExpr, ok := keyValue.Key.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok && ident.Name == "meta" {
					keys = append(keys, selectorExpr.Sel.Name)
				}
			}
		}
	}

	return keys
}

// extractEnumAnnotation extracts feature names from kubebuilder Enum annotation
func extractEnumAnnotation(filename string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	// Look for kubebuilder enum annotation in comments with strict boundary matching
	enumPattern := regexp.MustCompile(`^\s*//\s*\+kubebuilder:validation:Enum=([^;\s]+(?:;[^;\s]+)*)\s*$`)

	for _, commentGroup := range node.Comments {
		for _, comment := range commentGroup.List {
			matches := enumPattern.FindStringSubmatch(comment.Text)
			if len(matches) <= 1 {
				continue
			}
			enumValues := strings.Split(matches[1], ";")
			var features []string
			for _, value := range enumValues {
				value = strings.TrimSpace(value)
				if value != "" {
					features = append(features, value)
				}
			}
			sort.Strings(features)
			return features, nil
		}
	}

	// No enum annotation found - this is a warning condition, not an error
	return []string{}, nil
}

// displayResults shows what was found in each file
func displayResults(features, unreloadableKeys, enumFeatures []string) {
	fmt.Printf("Features defined in %s:\n", filepath.Base(featureFile))
	for _, feature := range features {
		fmt.Printf("  %s\n", feature)
	}
	fmt.Println()

	fmt.Printf("Features declared in unreloadable map in %s:\n", filepath.Base(reloadFile))
	for _, key := range unreloadableKeys {
		fmt.Printf("  %s\n", key)
	}
	fmt.Println()

	fmt.Printf("Features in kubebuilder Enum annotation:\n")
	for _, feature := range enumFeatures {
		fmt.Printf("  %s\n", feature)
	}
	fmt.Println()
}

// validateConsistency checks consistency between the three sources and returns exit code
func validateConsistency(features, unreloadableKeys, enumFeatures []string) int {
	exitCode := 0

	// Check features against unreloadable map
	if missing := difference(features, unreloadableKeys); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "ERROR: Missing features in unreloadable map in %s:\n", filepath.Base(reloadFile))
		for _, feature := range missing {
			fmt.Fprintf(os.Stderr, "  %s\n", feature)
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "To fix this, add entries to the unreloadable map like:")
		for _, feature := range missing {
			fmt.Fprintf(os.Stderr, "    meta.%s: {},\n", feature)
		}
		fmt.Fprintln(os.Stderr)
		exitCode = 1
	}

	if extra := difference(unreloadableKeys, features); len(extra) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: Extra features in unreloadable map in %s:\n", filepath.Base(reloadFile))
		for _, feature := range extra {
			fmt.Fprintf(os.Stderr, "  %s\n", feature)
		}
		fmt.Fprintln(os.Stderr, "Consider removing these if the features have been removed.")
		fmt.Fprintln(os.Stderr)
	}

	// Check features against enum annotation
	if missing := difference(features, enumFeatures); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "ERROR: Missing features in kubebuilder Enum annotation in %s:\n", filepath.Base(featureFile))
		for _, feature := range missing {
			fmt.Fprintf(os.Stderr, "  %s\n", feature)
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "To fix this, update the annotation to include missing features.")
		fmt.Fprintln(os.Stderr)
		exitCode = 1
	}

	if extra := difference(enumFeatures, features); len(extra) > 0 {
		fmt.Fprintf(os.Stderr, "ERROR: Extra features in kubebuilder Enum annotation in %s:\n", filepath.Base(featureFile))
		for _, feature := range extra {
			fmt.Fprintf(os.Stderr, "  %s\n", feature)
		}
		fmt.Fprintln(os.Stderr, "Remove these from the annotation if they are no longer defined.")
		fmt.Fprintln(os.Stderr)
		exitCode = 1
	}

	return exitCode
}

// difference returns elements in slice1 that are not in slice2
func difference(slice1, slice2 []string) []string {
	set2 := make(map[string]bool)
	for _, item := range slice2 {
		set2[item] = true
	}

	var result []string
	for _, item := range slice1 {
		if !set2[item] {
			result = append(result, item)
		}
	}
	return result
}
