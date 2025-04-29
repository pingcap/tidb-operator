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

package validation

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel/model"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/yaml"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/apiserver/pkg/cel/common"
)

var InstanceCRDs = []string{
	"crd/core.pingcap.com_pds.yaml",
	"crd/core.pingcap.com_tikvs.yaml",
	"crd/core.pingcap.com_tidbs.yaml",
	"crd/core.pingcap.com_tiflashes.yaml",
	"crd/core.pingcap.com_ticdcs.yaml",
	"crd/core.pingcap.com_tsos.yaml",
}

type Case struct {
	desc         string
	isCreate     bool
	current, old any
	wantErrs     []string
}

func Validate(t *testing.T, path string, cases []Case) {
	s := structuralSchemaFromCRD(t, path, "v1alpha1")

	validate := validateFunc(s)

	for i := range cases {
		c := &cases[i]

		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			ctx := context.Background()
			if c.isCreate {
				c.old = nil
			}
			errs := validate(ctx, nil, c.current, c.old)
			require.Equal(tt, len(c.wantErrs), len(errs), "%s: %v", c.desc, errs)
			for i, err := range errs {
				assert.EqualError(tt, err, c.wantErrs[i], c.desc)
			}
		})
	}
}

func validateFunc(s *schema.Structural) func(ctx context.Context, fieldPath *field.Path, obj, old any) field.ErrorList {
	schemaValidator := validation.NewSchemaValidatorFromOpenAPI(s.ToKubeOpenAPI())
	celValidator := cel.NewValidator(s, true, celconfig.RuntimeCELCostBudget)

	// validate openapi schema and cel
	return func(ctx context.Context, fieldPath *field.Path, obj, old any) field.ErrorList {
		var options []validation.ValidationOption
		var celOptions []cel.Option
		var correlatedObject *common.CorrelatedObject

		var errs field.ErrorList

		if old != nil {
			correlatedObject = common.NewCorrelatedObject(obj, old, &model.Structural{Structural: s})
			options = append(options, validation.WithRatcheting(correlatedObject))
			celOptions = append(celOptions, cel.WithRatcheting(correlatedObject))
			errs = append(errs, validation.ValidateCustomResourceUpdate(fieldPath, obj, old, schemaValidator, options...)...)
		} else {
			errs = append(errs, validation.ValidateCustomResource(fieldPath, obj, schemaValidator)...)
		}

		if has := hasBlockingErr(errs); !has {
			err, _ := celValidator.Validate(ctx, fieldPath, s, obj, old, celconfig.RuntimeCELCostBudget, celOptions...)
			errs = append(errs, err...)
		}

		return errs
	}
}

// Copied from k8s.io/apiextensions-apiserver/pkg/registry/customresource/strategy.go
// NOTE: DO NOT depend on this function in test.
//
// OpenAPIv3 type/maxLength/maxItems/MaxProperties/required/enum violation/wrong type field validation failures are viewed as blocking err for CEL validation
func hasBlockingErr(errs field.ErrorList) bool {
	for _, err := range errs {
		if err.Type == field.ErrorTypeNotSupported || err.Type == field.ErrorTypeRequired || err.Type == field.ErrorTypeTooLong || err.Type == field.ErrorTypeTooMany || err.Type == field.ErrorTypeTypeInvalid {
			return true
		}
	}
	return false
}

func structuralSchemaFromCRD(t *testing.T, crdPath, version string) *schema.Structural {
	data, err := os.ReadFile(crdPath)
	require.NoError(t, err)

	var crd apiextensionsv1.CustomResourceDefinition
	err = yaml.Unmarshal(data, &crd)
	require.NoError(t, err)

	for _, v := range crd.Spec.Versions {
		if v.Name != version {
			continue
		}

		var internalSchema apiextensions.JSONSchemaProps
		err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(v.Schema.OpenAPIV3Schema, &internalSchema, nil)
		require.NoError(t, err, "failed to convert JSONSchemaProps for crd %s version %s: %v", crdPath, v.Name, err)
		s, err := schema.NewStructural(&internalSchema)
		require.NoError(t, err, "failed to create StructuralSchema for crd %s version %s: %v", crdPath, v.Name, err)
		return s
	}

	require.FailNowf(t, "failed to find crd %s version %s", crdPath, version)

	return nil
}
