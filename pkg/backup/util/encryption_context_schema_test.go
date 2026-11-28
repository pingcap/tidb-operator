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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdvalidation "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func TestS3SSEOpenAPIDeprecationParagraph(t *testing.T) {
	definitions := v1alpha1.GetOpenAPIDefinitions(func(string) spec.Ref { return spec.Ref{} })
	s3 := definitions["github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.S3StorageProvider"].Schema
	require.Contains(t, s3.Properties["sse"].Description, "\n\nDeprecated: use ServerSideEncryption.Algorithm instead.")
	require.Contains(t, s3.Properties["sseKmsEncryptionContext"].Description, "\n\nDeprecated: use ServerSideEncryption.KMSEncryptionContext instead.")
}

// Check the shipped schemas, including every occurrence of the shared S3 type.
func TestS3ServerSideEncryptionCRDSchema(t *testing.T) {
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

			// Checking the whole CRD covers CEL's static budget in the unbounded
			// Federation VolumeRestore clusters array. Singleton-list membership
			// compares values once and keeps the Kubernetes cost estimate within budget.
			apiextensionsv1.SetDefaults_CustomResourceDefinition(&crd)
			var internalCRD apiextensions.CustomResourceDefinition
			require.NoError(t, apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(&crd, &internalCRD, nil))
			require.Empty(t, crdvalidation.ValidateCustomResourceDefinition(t.Context(), &internalCRD))

			found := 0
			var walk func(apiextensionsv1.JSONSchemaProps)
			walk = func(schema apiextensionsv1.JSONSchemaProps) {
				// The legacy sse field identifies each occurrence independently of the new schema.
				if _, ok := schema.Properties["sse"]; ok {
					found++
					require.Contains(t, schema.Properties, "sseKmsEncryptionContext")
					require.Contains(t, schema.Properties, "serverSideEncryption")
					require.Len(t, schema.XValidations, 2)
					var internal apiextensions.JSONSchemaProps
					require.NoError(t, apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(&schema, &internal, nil))
					testS3EncryptionCEL(t, &internal)
					validator, _, err := validation.NewSchemaValidator(&internal)
					require.NoError(t, err)
					for _, test := range []struct {
						json  string
						valid bool
					}{
						{`{"provider":"aws"}`, true},
						{`{"provider":"aws","serverSideEncryption":{}}`, true},
						{`{"provider":"aws","serverSideEncryption":{"algorithm":"AES256"}}`, true},
						{`{"provider":"aws","serverSideEncryption":{"kmsKeyId":"alias/backup"}}`, true},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":{}}}}`, true},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":{"tenant":"租户"}}}}`, true},
						{`{"provider":"aws","serverSideEncryption":{"algorithm":"aws:kms","kmsKeyId":"alias/backup","kmsEncryptionContext":{"values":{"tenant":"租户"}}}}`, true},
						{`{"provider":"aws","sse":"aws:kms","serverSideEncryption":{"algorithm":"aws:kms"}}`, true},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{}}}`, true},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":"租户"}}}`, true},
						{`{"provider":"aws","sseKmsEncryptionContext":"invalid"}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":null}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":[]}}`, false},
						{`{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":123}}}`, false},
						{`{"provider":"aws","serverSideEncryption":true}`, false},
						{`{"provider":"aws","serverSideEncryption":{"algorithm":123}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsKeyId":true}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{}}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":null}}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":{"tenant":123}}}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":{"tenant":true}}}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":{"tenant":{"nested":"value"}}}}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":{"tenant":[]}}}}`, false},
						{`{"provider":"aws","serverSideEncryption":{"kmsEncryptionContext":{"values":[]}}}`, false},
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

func testS3EncryptionCEL(t *testing.T, schema *apiextensions.JSONSchemaProps) {
	t.Helper()
	structural, err := structuralschema.NewStructural(schema)
	require.NoError(t, err)
	validator := cel.NewValidator(structural, false, celconfig.PerCallLimit)
	require.NotNil(t, validator)
	for _, tc := range []struct {
		json    string
		message string
	}{
		{json: `{"provider":"aws","sse":"aws:kms","serverSideEncryption":{"algorithm":"aws:kms"}}`},
		{
			json:    `{"provider":"aws","sse":"aws:kms","serverSideEncryption":{"algorithm":"AES256"}}`,
			message: "s3.sse conflicts with s3.serverSideEncryption.algorithm",
		},
		{json: `{"provider":"aws","sse":"","serverSideEncryption":{"algorithm":"aws:kms"}}`},
		{json: `{"provider":"aws","sse":"aws:kms","serverSideEncryption":{"algorithm":""}}`},
		{json: `{"provider":"aws","sse":"aws:kms","serverSideEncryption":{"kmsKeyId":"alias/backup"}}`},
		{json: `{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":"value"}},"serverSideEncryption":{"algorithm":"aws:kms"}}`},
		{json: `{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":"value"}},"serverSideEncryption":{"kmsEncryptionContext":{"values":{"tenant":"value"}}}}`},
		{json: `{"provider":"aws","sseKmsEncryptionContext":{"values":{}},"serverSideEncryption":{"kmsEncryptionContext":{"values":{}}}}`},
		{
			json:    `{"provider":"aws","sseKmsEncryptionContext":{"values":{"tenant":"legacy-value"}},"serverSideEncryption":{"kmsEncryptionContext":{"values":{"tenant":"nested-value"}}}}`,
			message: "s3.sseKmsEncryptionContext conflicts with s3.serverSideEncryption.kmsEncryptionContext",
		},
		{
			json:    `{"provider":"aws","sseKmsEncryptionContext":{"values":{"legacy":""}},"serverSideEncryption":{"kmsEncryptionContext":{"values":{"nested":""}}}}`,
			message: "s3.sseKmsEncryptionContext conflicts with s3.serverSideEncryption.kmsEncryptionContext",
		},
	} {
		var object map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(tc.json), &object))
		errs, _ := validator.Validate(t.Context(), field.NewPath("s3"), structural, object, nil, celconfig.RuntimeCELCostBudget)
		if tc.message == "" {
			require.Empty(t, errs, "%s: %v", tc.json, errs)
		} else {
			require.Len(t, errs, 1, "%s: %v", tc.json, errs)
			require.Equal(t, tc.message, errs[0].Detail)
		}
	}
}
