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

package crd

import (
	"errors"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestCheckVersion(t *testing.T) {
	cases := []struct {
		name        string
		oldVer      string
		newVer      string
		isDirty     bool
		wantChanged bool
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "same version, not dirty",
			oldVer:      "1.0.0",
			newVer:      "1.0.0",
			isDirty:     false,
			wantChanged: false,
			wantErr:     false,
		},
		{
			name:        "same version, dirty",
			oldVer:      "1.0.0",
			newVer:      "1.0.0",
			isDirty:     true,
			wantChanged: true,
			wantErr:     false,
		},
		{
			name:        "upgrade version",
			oldVer:      "1.0.0",
			newVer:      "1.1.0",
			isDirty:     false,
			wantChanged: true,
			wantErr:     false,
		},
		{
			name:        "upgrade major version",
			oldVer:      "1.0.0",
			newVer:      "2.0.0",
			isDirty:     false,
			wantChanged: true,
			wantErr:     false,
		},
		{
			name:        "downgrade version",
			oldVer:      "2.0.0",
			newVer:      "1.0.0",
			isDirty:     false,
			wantChanged: false,
			wantErr:     true,
			errMsg:      "cannot downgrade version from 2.0.0 to 1.0.0",
		},
		{
			name:        "invalid old version",
			oldVer:      "invalid",
			newVer:      "1.0.0",
			isDirty:     false,
			wantChanged: false,
			wantErr:     true,
			errMsg:      "old version invalid is invalid",
		},
		{
			name:        "invalid new version",
			oldVer:      "1.0.0",
			newVer:      "invalid",
			isDirty:     false,
			wantChanged: false,
			wantErr:     true,
			errMsg:      "new version invalid is invalid",
		},
		{
			name:        "prerelease version upgrade",
			oldVer:      "1.0.0-alpha",
			newVer:      "1.0.0",
			isDirty:     false,
			wantChanged: true,
			wantErr:     false,
		},
		{
			name:        "patch version upgrade",
			oldVer:      "1.0.0",
			newVer:      "1.0.1",
			isDirty:     false,
			wantChanged: true,
			wantErr:     false,
		},
		{
			name:        "prerelease to release",
			oldVer:      "1.0.0-beta.1",
			newVer:      "1.0.0",
			isDirty:     false,
			wantChanged: true,
			wantErr:     false,
		},
		{
			name:        "with build metadata",
			oldVer:      "1.0.0+20130313144700",
			newVer:      "1.0.1",
			isDirty:     false,
			wantChanged: true,
			wantErr:     false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.name, func(tt *testing.T) {
			tt.Parallel()
			changed, err := checkVersion(c.oldVer, c.newVer, c.isDirty)
			if c.wantErr {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), c.errMsg)
			} else {
				require.NoError(tt, err)
				assert.Equal(tt, c.wantChanged, changed)
			}
		})
	}
}

func TestGetCurrentCRD(t *testing.T) {
	cases := []struct {
		name     string
		crdName  string
		existing *apiextensionsv1.CustomResourceDefinition
		mockErr  error
		wantCRD  bool
		wantErr  bool
	}{
		{
			name:    "CRD exists",
			crdName: "test.example.com",
			existing: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test.example.com",
				},
			},
			wantCRD: true,
			wantErr: false,
		},
		{
			name:     "CRD does not exist",
			crdName:  "notfound.example.com",
			existing: nil,
			wantCRD:  false,
			wantErr:  false,
		},
		{
			name:    "Get returns error (not NotFound)",
			crdName: "error.example.com",
			mockErr: errors.New("internal error"),
			wantCRD: false,
			wantErr: true,
		},
		{
			name:     "CRD not found returns nil without error",
			crdName:  "notfound.example.com",
			existing: nil,
			wantCRD:  false,
			wantErr:  false,
		},
		{
			name:    "Get returns unauthorized error",
			crdName: "test.example.com",
			mockErr: apierrors.NewUnauthorized("unauthorized"),
			wantCRD: false,
			wantErr: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.name, func(tt *testing.T) {
			tt.Parallel()
			var objs []client.Object
			if c.existing != nil {
				objs = append(objs, c.existing)
			}
			fc := client.NewFakeClient(objs...)
			if c.mockErr != nil {
				fc.WithError("get", "customresourcedefinitions", c.mockErr)
			}

			crd, err := getCurrentCRD(tt.Context(), fc, c.crdName)

			if c.wantErr {
				require.Error(tt, err)
			} else {
				require.NoError(tt, err)
				if c.wantCRD {
					require.NotNil(tt, crd)
					assert.Equal(tt, c.crdName, crd.Name)
				} else {
					assert.Nil(tt, crd)
				}
			}
		})
	}
}

func TestApplyCRD(t *testing.T) {
	sampleCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test.example.com
spec:
  group: example.com
  names:
    kind: Test
    plural: tests
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
`

	sampleCRDWithAnnotation := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test.example.com
  annotations:
    existing-annotation: existing-value
spec:
  group: example.com
  names:
    kind: Test
    plural: tests
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
`

	cases := []struct {
		name             string
		fileName         string
		fileContent      string
		existingCRD      *apiextensionsv1.CustomResourceDefinition
		version          string
		allowEmptyOld    bool
		isDirty          bool
		mockGetError     error
		wantErr          bool
		errMsg           string
		checkAnnotations bool
	}{
		{
			name:        "apply new CRD",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			version:     "1.0.0",
			wantErr:     false,
		},
		{
			name:        "apply CRD with existing version (no change needed)",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			existingCRD: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test.example.com",
					Annotations: map[string]string{
						versionAnnoKey: "1.0.0",
					},
				},
			},
			version: "1.0.0",
			isDirty: false,
			wantErr: false,
		},
		{
			name:        "apply CRD with version upgrade",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			existingCRD: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test.example.com",
					Annotations: map[string]string{
						versionAnnoKey: "1.0.0",
					},
				},
			},
			version: "1.1.0",
			wantErr: false,
		},
		{
			name:        "apply CRD with dirty flag (same version)",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			existingCRD: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test.example.com",
					Annotations: map[string]string{
						versionAnnoKey: "1.0.0",
					},
				},
			},
			version: "1.0.0",
			isDirty: true,
			wantErr: false,
		},
		{
			name:        "existing CRD without version annotation, allow empty",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			existingCRD: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test.example.com",
				},
			},
			version:       "1.0.0",
			allowEmptyOld: true,
			wantErr:       false,
		},
		{
			name:        "existing CRD without version annotation, disallow empty",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			existingCRD: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test.example.com",
				},
			},
			version:       "1.0.0",
			allowEmptyOld: false,
			wantErr:       true,
			errMsg:        "cannot find old version from crd test.example.com",
		},
		{
			name:        "invalid YAML file",
			fileName:    "invalid.yaml",
			fileContent: "invalid: yaml: content:",
			version:     "1.0.0",
			wantErr:     true,
			errMsg:      "cannot unmarshal yaml",
		},
		{
			name:        "non-yaml file should be skipped at ApplyCRDs level",
			fileName:    "test.txt",
			fileContent: "not a yaml file",
			version:     "1.0.0",
			wantErr:     true,
		},
		{
			name:         "Get returns error during CRD retrieval",
			fileName:     "test.yaml",
			fileContent:  sampleCRD,
			version:      "1.0.0",
			mockGetError: fmt.Errorf("connection timeout"),
			wantErr:      true,
			errMsg:       "cannot get crd test.example.com",
		},
		{
			name:        "version downgrade error",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			existingCRD: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test.example.com",
					Annotations: map[string]string{
						versionAnnoKey: "2.0.0",
					},
				},
			},
			version: "1.0.0",
			wantErr: true,
			errMsg:  "cannot downgrade version",
		},
		{
			name:        "file not found error",
			fileName:    "notfound.yaml",
			fileContent: "",
			version:     "1.0.0",
			wantErr:     true,
			errMsg:      "cannot read file",
		},
		{
			name:             "apply CRD preserves existing annotations",
			fileName:         "test.yaml",
			fileContent:      sampleCRDWithAnnotation,
			version:          "1.0.0",
			wantErr:          false,
			checkAnnotations: true,
		},
		{
			name:        "apply CRD adds annotation to empty map",
			fileName:    "test.yaml",
			fileContent: sampleCRD,
			version:     "1.0.0",
			wantErr:     false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.name, func(tt *testing.T) {
			tt.Parallel()

			// Create test filesystem
			fsys := fstest.MapFS{}
			if c.fileContent != "" {
				fsys["crd/"+c.fileName] = &fstest.MapFile{
					Data: []byte(c.fileContent),
				}
			}

			var objs []client.Object
			if c.existingCRD != nil {
				objs = append(objs, c.existingCRD)
			}
			fc := client.NewFakeClient(objs...)
			if c.mockGetError != nil {
				fc.WithError("get", "customresourcedefinitions", c.mockGetError)
			}

			cfg := &Config{
				Client:               fc,
				Version:              c.version,
				AllowEmptyOldVersion: c.allowEmptyOld,
				IsDirty:              c.isDirty,
				CRDDir:               fsys,
			}

			err := applyCRD(tt.Context(), cfg, c.fileName)

			if c.wantErr {
				require.Error(tt, err)
				if c.errMsg != "" {
					assert.Contains(tt, err.Error(), c.errMsg)
				}
			} else {
				require.NoError(tt, err)

				// Verify the CRD was applied with correct version annotation
				var crd apiextensionsv1.CustomResourceDefinition
				err := fc.Get(tt.Context(), client.ObjectKey{Name: "test.example.com"}, &crd)
				require.NoError(tt, err)
				assert.Equal(tt, c.version, crd.Annotations[versionAnnoKey])
				if c.checkAnnotations {
					assert.Equal(tt, "existing-value", crd.Annotations["existing-annotation"])
				}
			}
		})
	}
}

func TestApplyCRDs(t *testing.T) {
	sampleCRD1 := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test1.example.com
spec:
  group: example.com
  names:
    kind: Test1
    plural: test1s
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
`

	sampleCRD2 := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test2.example.com
spec:
  group: example.com
  names:
    kind: Test2
    plural: test2s
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
`

	cases := []struct {
		name        string
		files       map[string]string
		version     string
		mockError   error
		mockErrorOp string
		wantErr     bool
		wantCount   int
	}{
		{
			name: "apply multiple CRDs",
			files: map[string]string{
				"crd/test1.yaml": sampleCRD1,
				"crd/test2.yaml": sampleCRD2,
			},
			version:   "1.0.0",
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "skip non-yaml files",
			files: map[string]string{
				"crd/test1.yaml":  sampleCRD1,
				"crd/test2.yaml":  sampleCRD2,
				"crd/readme.txt":  "not a yaml file",
				"crd/config.json": "{}",
			},
			version:   "1.0.0",
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "empty directory",
			files: map[string]string{
				"crd/.keep": "",
			},
			version:   "1.0.0",
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "one invalid CRD stops processing",
			files: map[string]string{
				"crd/test1.yaml":   sampleCRD1,
				"crd/invalid.yaml": "this is not\n  valid: yaml: content: [unclosed",
			},
			version: "1.0.0",
			wantErr: true,
		},
		{
			name: "client patch error",
			files: map[string]string{
				"crd/test.yaml": sampleCRD1,
			},
			version:     "1.0.0",
			mockError:   apierrors.NewInternalError(errors.New("server error")),
			mockErrorOp: "patch",
			wantErr:     true,
		},
		{
			name: "directory not found error",
			files: map[string]string{
				"other/file.txt": "content",
			},
			version: "1.0.0",
			wantErr: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.name, func(tt *testing.T) {
			tt.Parallel()

			// Create test filesystem
			fsys := fstest.MapFS{}
			for path, content := range c.files {
				fsys[path] = &fstest.MapFile{
					Data: []byte(content),
				}
			}

			fc := client.NewFakeClient()
			if c.mockError != nil {
				fc.WithError(c.mockErrorOp, "customresourcedefinitions", c.mockError)
			}

			cfg := &Config{
				Client:               fc,
				Version:              c.version,
				AllowEmptyOldVersion: true,
				IsDirty:              false,
				CRDDir:               fsys,
			}

			err := ApplyCRDs(tt.Context(), cfg)

			if c.wantErr {
				require.Error(tt, err)
			} else {
				require.NoError(tt, err)

				// Verify the expected number of CRDs were applied
				var crdList apiextensionsv1.CustomResourceDefinitionList
				err := fc.List(tt.Context(), &crdList)
				require.NoError(tt, err)
				assert.Len(tt, crdList.Items, c.wantCount)

				// Verify all CRDs have the version annotation
				for _, crd := range crdList.Items {
					assert.Equal(tt, c.version, crd.Annotations[versionAnnoKey])
				}
			}
		})
	}
}
