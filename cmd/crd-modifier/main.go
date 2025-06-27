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

//nolint:gosec
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	kyaml "sigs.k8s.io/yaml"
)

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "manifests/crd", "dir of crds")
	flag.Parse()

	files, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Errorf("failed to read dir %s: %w", dir, err))
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".yaml") {
			continue
		}

		path := filepath.Join(dir, fileName)
		crd, err := loadCRD(path)
		if err != nil {
			panic(err)
		}

		if len(crd.Spec.Versions) == 0 {
			continue
		}
		schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema

		if checkPathExists(schema, "spec", "template", "spec", "overlay") {
			// It's a group CRD, which has an overlay in spec.template.spec.overlay
			removeUselessRequired(crd, "spec", "template", "spec", "overlay", "pod")
		} else if checkPathExists(schema, "spec", "overlay") {
			// It's an instance CRD, which has an overlay in spec.overlay
			removeUselessRequired(crd, "spec", "overlay", "pod")
		} else {
			continue
		}

		if err := saveCRD(path, crd); err != nil {
			panic(err)
		}
	}
}

// checkPathExists checks if a path of keys exists in the schema's properties.
func checkPathExists(schema *apiextensionsv1.JSONSchemaProps, keys ...string) bool {
	current := schema
	for _, key := range keys {
		if current == nil {
			return false
		}
		child, ok := current.Properties[key]
		if !ok {
			return false
		}
		current = &child
	}
	return true
}

func saveCRD(path string, crd *apiextensionsv1.CustomResourceDefinition) error {
	bytes, err := kyaml.Marshal(crd)
	if err != nil {
		return err
	}
	obj := map[string]any{}
	if err := kyaml.Unmarshal(bytes, &obj); err != nil {
		return err
	}
	delete(obj, "status")
	metadata := obj["metadata"].(map[string]any)
	delete(metadata, "creationTimestamp")

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	data, err := kyaml.Marshal(&obj)
	if err != nil {
		return err
	}
	if _, err := f.WriteString("---\n"); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}

	return nil
}

func loadCRD(path string) (*apiextensionsv1.CustomResourceDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %w", path, err)
	}

	var crd apiextensionsv1.CustomResourceDefinition
	if err := yamlutil.Unmarshal(data, &crd); err != nil {
		return nil, fmt.Errorf("cannot unmarshal yaml: %w", err)
	}

	return &crd, nil
}

func removeUselessRequired(crd *apiextensionsv1.CustomResourceDefinition, keys ...string) {
	for i := range crd.Spec.Versions {
		v := &crd.Spec.Versions[i]
		handleVersion(v, keys...)
	}
}

func handleVersion(version *apiextensionsv1.CustomResourceDefinitionVersion, keys ...string) {
	handleSelectedJSONSchemaProps(version.Schema.OpenAPIV3Schema, keys...)
}

func handleSelectedJSONSchemaProps(props *apiextensionsv1.JSONSchemaProps, keys ...string) {
	if len(keys) == 0 {
		// When the target path is reached, call handleJSONSchemaProps to remove the required fields.
		// `nil` is passed for retainKeys, which means no keys should be retained (i.e., all are removed).
		handleJSONSchemaProps(props, nil)
		return
	}

	key := keys[0]
	child, ok := props.Properties[key]
	if !ok {
		// This case should ideally not be reached if the CRD structure is as expected.
		fmt.Println("cannot find keys: ", keys)
		panic("no props")
	}
	// Recurse into the next level of the schema.
	handleSelectedJSONSchemaProps(&child, keys[1:]...)
	props.Properties[key] = child
}

// handleJSONSchemaProps recursively removes the `required` constraint from an object property.
// It works by replacing the `Required` slice with the intersection of itself and `retainKeys`.
// When `retainKeys` is nil or empty, the intersection is always an empty slice, effectively
// clearing all `required` constraints.
func handleJSONSchemaProps(props *apiextensionsv1.JSONSchemaProps, retainKeys []string) {
	switch props.Type {
	case "object":
		switch {
		case props.XMapType == nil || *props.XMapType == "granular":
			// For granular maps, remove the `required` fields.
			// The call to intersection with a `nil` retainKeys will clear the list.
			props.Required = intersection(props.Required, retainKeys)
			// Recursively process all child properties to ensure nested objects are also handled.
			//nolint:gocritic
			for key, val := range props.Properties {
				handleJSONSchemaProps(&val, nil)
				props.Properties[key] = val
			}
		case *props.XMapType == "atomic":
			// Do not modify atomic maps.
			return
		}
	case "array":
		switch {
		case props.XListType == nil:
			return
		case *props.XListType == "atomic":
			return
		case *props.XListType == "set":
			return
		case *props.XListType == "map":
			// For map-type lists, process the schema of the list items.
			handleJSONSchemaPropsOrArray(props.Items, props.XListMapKeys)
		}
	}
}

// handleJSONSchemaPropsOrArray handles schemas that can be a single schema or an array of schemas.
func handleJSONSchemaPropsOrArray(props *apiextensionsv1.JSONSchemaPropsOrArray, retainKeys []string) {
	if props.Schema != nil {
		handleJSONSchemaProps(props.Schema, retainKeys)
	}
	for i := range props.JSONSchemas {
		s := &props.JSONSchemas[i]
		handleJSONSchemaProps(s, retainKeys)
	}
}

// intersection returns a new slice containing only the elements that exist in both input slices.
// If slice `b` is nil or empty, it will always return an empty slice.
// This is the mechanism used to clear the `required` fields from a schema property.
func intersection(a, b []string) []string {
	var ret []string
	for _, key := range a {
		for _, retain := range b {
			if key == retain {
				ret = append(ret, key)
			}
		}
	}

	return ret
}
