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
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Check the shipped schemas, including every occurrence of the shared S3 type.
func TestS3EncryptionContextCRDSchema(t *testing.T) {
	for file, count := range map[string]int{
		"v1/pingcap.com_backups.yaml":                                     1,
		"v1/pingcap.com_backupschedules.yaml":                             4,
		"v1/pingcap.com_restores.yaml":                                    2,
		"v1/pingcap.com_compactbackups.yaml":                              1,
		"federation/v1/federation.pingcap.com_volumebackups.yaml":         1,
		"federation/v1/federation.pingcap.com_volumebackupschedules.yaml": 1,
		"federation/v1/federation.pingcap.com_volumerestores.yaml":        1,
	} {
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("../../../manifests/crd", file))
			require.NoError(t, err)
			var crd apiextensionsv1.CustomResourceDefinition
			require.NoError(t, yaml.Unmarshal(content, &crd))
			found := 0
			var walk func(apiextensionsv1.JSONSchemaProps)
			walk = func(schema apiextensionsv1.JSONSchemaProps) {
				if _, ok := schema.Properties["sseKmsEncryptionContext"]; ok {
					found++
					var internal apiextensions.JSONSchemaProps
					require.NoError(t, apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&schema, &internal, nil))
					validator, _, err := validation.NewSchemaValidator(&internal)
					require.NoError(t, err)
					for _, test := range []struct {
						json  string
						valid bool
					}{
						{`{"provider":"aws"}`, true},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{}}}`, true},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":"租户"}}}`, true},
						{`{"provider":"aws","sseKmsEncryptionContext":{}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":null}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":123}}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":true}}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":{"nested":"value"}}}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":[]}}`, false},
					} {
						var object interface{}
						require.NoError(t, json.Unmarshal([]byte(test.json), &object))
						errs := validation.ValidateCustomResource(field.NewPath("s3"), object, validator)
						require.Equal(t, test.valid, len(errs) == 0, "%s: %v", test.json, errs)
					}
				}
				for _, property := range schema.Properties {
					walk(property)
				}
				if schema.Items != nil && schema.Items.Schema != nil {
					walk(*schema.Items.Schema)
				}
			}
			for _, version := range crd.Spec.Versions {
				walk(*version.Schema.OpenAPIV3Schema)
			}
			require.Equal(t, count, found)
		})
	}
}
