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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	kyaml "sigs.k8s.io/yaml"
)

var (
	groupCRDs = []string{
		"core.pingcap.com_pdgroups.yaml",
		"core.pingcap.com_tikvgroups.yaml",
		"core.pingcap.com_tidbgroups.yaml",
		"core.pingcap.com_tiflashgroups.yaml",
		"core.pingcap.com_ticdcgroups.yaml",
	}
	instanceCRDs = []string{
		"core.pingcap.com_pds.yaml",
		"core.pingcap.com_tikvs.yaml",
		"core.pingcap.com_tidbs.yaml",
		"core.pingcap.com_tiflashes.yaml",
		"core.pingcap.com_ticdcs.yaml",
	}
)

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "manifests/crd", "dir of crds")
	flag.Parse()
	for _, crdPath := range groupCRDs {
		path := filepath.Join(dir, crdPath)
		crd, err := loadCRD(path)
		if err != nil {
			panic(err)
		}

		removeUselessRequired(crd, "spec", "template", "spec", "overlay", "pod")
		if err := saveCRD(path, crd); err != nil {
			panic(err)
		}
	}
	for _, crdPath := range instanceCRDs {
		path := filepath.Join(dir, crdPath)
		crd, err := loadCRD(path)
		if err != nil {
			panic(err)
		}

		removeUselessRequired(crd, "spec", "overlay", "pod")
		if err := saveCRD(path, crd); err != nil {
			panic(err)
		}
	}
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
		handleJSONSchemaProps(props, nil)
		return
	}

	key := keys[0]
	child, ok := props.Properties[key]
	if !ok {
		fmt.Println("cannot find keys: ", keys)
		panic("no props")
	}
	handleSelectedJSONSchemaProps(&child, keys[1:]...)
	props.Properties[key] = child
}

func handleJSONSchemaProps(props *apiextensionsv1.JSONSchemaProps, retainKeys []string) {
	switch props.Type {
	case "object":
		switch {
		case props.XMapType == nil || *props.XMapType == "granular":
			// remove required for granular map
			props.Required = intersection(props.Required, retainKeys)
			for key, val := range props.Properties {
				handleJSONSchemaProps(&val, nil)
				props.Properties[key] = val
			}
		case *props.XMapType == "atomic":
			return
		}
	case "array":
		switch {
		case props.XListType == nil:
			// panic("unknown list type for array: " + props.Description)
			return
		case *props.XListType == "atomic":
			return
		case *props.XListType == "set":
			return
		case *props.XListType == "map":
			handleJSONSchemaPropsOrArray(props.Items, props.XListMapKeys)
		}
	}
}

func handleJSONSchemaPropsOrArray(props *apiextensionsv1.JSONSchemaPropsOrArray, retainKeys []string) {
	if props.Schema != nil {
		handleJSONSchemaProps(props.Schema, retainKeys)
	}
	for i := range props.JSONSchemas {
		s := &props.JSONSchemas[i]
		handleJSONSchemaProps(s, retainKeys)
	}
}

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
