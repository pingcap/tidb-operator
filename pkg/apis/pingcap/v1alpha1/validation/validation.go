// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License a
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"encoding/json"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateTidbCluster validates a TidbCluster.
func ValidateTidbCluster(tc *v1alpha1.TidbCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// validate metadata
	fldPath := field.NewPath("metadata")
	// validate metadata/annotations
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(tc.ObjectMeta.Annotations, fldPath.Child("annotations"))...)
	for _, key := range []string{label.AnnPDDeleteSlots, label.AnnTiDBDeleteSlots, label.AnnTiKVDeleteSlots} {
		allErrs = append(allErrs, validateDeleteSlots(tc.ObjectMeta.Annotations, key, fldPath.Child("annotations", key))...)
	}
	// validate spec
	allErrs = append(allErrs, ValidateTidbClusterSpec(&tc.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateDeleteSlots(annotations map[string]string, key string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if annotations != nil {
		if value, ok := annotations[key]; ok {
			var slice []int
			err := json.Unmarshal([]byte(value), &slice)
			if err != nil {
				msg := fmt.Sprintf("value of %q annotation must be a JSON list of integer", key)
				allErrs = append(allErrs, field.Invalid(fldPath, value, msg))
			}
		}
	}
	return allErrs
}

// ValidateTidbClusterSpec validates a TidbClusterSpec
func ValidateTidbClusterSpec(spec *v1alpha1.TidbClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO
	return allErrs
}
