// Copyright 2019 PingCAP, Inc.
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

package validation

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	v1 "k8s.io/api/core/v1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateTidbCluster validates a TidbCluster, it performs basic validation for all TidbClusters despite it is legacy
// or not
func ValidateTidbCluster(tc *v1alpha1.TidbCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// validate metadata
	fldPath := field.NewPath("metadata")
	// validate metadata/annotations
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(tc.ObjectMeta.Annotations, fldPath.Child("annotations"))...)
	for _, key := range []string{label.AnnPDDeleteSlots, label.AnnTiDBDeleteSlots, label.AnnTiKVDeleteSlots} {
		allErrs = append(allErrs, validateDeleteSlots(tc.ObjectMeta.Annotations, key, fldPath.Child("annotations", key))...)
	}
	return allErrs
}

// ValidateCreateTidbCLuster validates a newly created TidbCluster
func ValidateCreateTidbCluster(tc *v1alpha1.TidbCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// basic validation
	allErrs = append(allErrs, ValidateTidbCluster(tc)...)
	allErrs = append(allErrs, validateNewTidbClusterSpec(&tc.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateUpdateTidbCluster validates a new TidbCluster against an existing TidbCluster to be updated
func ValidateUpdateTidbCluster(old, tc *v1alpha1.TidbCluster) field.ErrorList {

	allErrs := field.ErrorList{}
	// basic validation
	allErrs = append(allErrs, ValidateTidbCluster(tc)...)
	if old.GetInstanceName() != tc.GetInstanceName() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("labels"), tc.Labels,
			"The instance must not be mutate or set value other than the cluster name"))
	}
	allErrs = append(allErrs, validateUpdatePDConfig(old.Spec.PD.Config, tc.Spec.PD.Config, field.NewPath("spec.pd.config"))...)
	allErrs = append(allErrs, disallowUsingLegacyAPIInNewCluster(old, tc)...)

	return allErrs
}

// For now we limit some validations only in Create phase to keep backward compatibility
// TODO(aylei): call this in ValidateTidbCluster after we deprecated the old versions of helm chart officially
func validateNewTidbClusterSpec(spec *v1alpha1.TidbClusterSpec, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.Version == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("version"), spec.Version, "version must not be empty"))
	}
	if spec.TiDB.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.baseImage"), spec.TiDB.BaseImage, "baseImage of TiDB must not be empty"))
	}
	if spec.PD.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.baseImage"), spec.PD.BaseImage, "baseImage of PD must not be empty"))
	}
	if spec.TiKV.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.baseImage"), spec.TiKV.BaseImage, "baseImage of TiKV must not be empty"))
	}
	if spec.TiDB.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.image"), spec.TiDB.Image, "image has been deprecated, use baseImage instead"))
	}
	if spec.TiKV.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.image"), spec.TiKV.Image, "image has been deprecated, use baseImage instead"))
	}
	if spec.PD.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.image"), spec.PD.Image, "image has been deprecated, use baseImage instead"))
	}
	if _, ok := spec.PD.ResourceRequirements.Requests[v1.ResourceStorage]; !ok {
		allErrs = append(allErrs, field.Required(path.Child("tikv.resources.requests").Key((string(v1.ResourceStorage))), "request storage of PD must not be empty"))
	}
	if _, ok := spec.TiKV.ResourceRequirements.Requests[v1.ResourceStorage]; !ok {
		allErrs = append(allErrs, field.Required(path.Child("tikv.resources.requests").Key((string(v1.ResourceStorage))), "request storage of TiKV must not be empty"))
	}
	return allErrs
}

// disallowUsingLegacyAPIInNewCluster checks if user use the legacy API in newly create cluster during update
// TODO(aylei): this could be removed after we enable validateTidbCluster() in update, which is more strict
func disallowUsingLegacyAPIInNewCluster(old, tc *v1alpha1.TidbCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	path := field.NewPath("spec")
	if old.Spec.Version != "" && tc.Spec.Version == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("version"), tc.Spec.Version, "version must not be empty"))
	}
	if old.Spec.TiDB.BaseImage != "" && tc.Spec.TiDB.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.baseImage"), tc.Spec.TiDB.BaseImage, "baseImage of TiDB must not be empty"))
	}
	if old.Spec.PD.BaseImage != "" && tc.Spec.PD.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.baseImage"), tc.Spec.PD.BaseImage, "baseImage of PD must not be empty"))
	}
	if old.Spec.TiKV.BaseImage != "" && tc.Spec.TiKV.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.baseImage"), tc.Spec.TiKV.BaseImage, "baseImage of TiKV must not be empty"))
	}
	if old.Spec.TiDB.Config != nil && tc.Spec.TiDB.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.config"), tc.Spec.TiDB.Config, "tidb.config must not be nil"))
	}
	if old.Spec.TiKV.Config != nil && tc.Spec.TiKV.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.config"), tc.Spec.TiKV.Config, "TiKV.config must not be nil"))
	}
	if old.Spec.PD.Config != nil && tc.Spec.PD.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.config"), tc.Spec.PD.Config, "PD.config must not be nil"))
	}
	return allErrs
}

func validateUpdatePDConfig(old, conf *v1alpha1.PDConfig, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// for newly created cluster, both old and new are non-nil, guaranteed by validation
	if old == nil || conf == nil {
		return allErrs
	}
	if !reflect.DeepEqual(old.Schedule, conf.Schedule) {
		allErrs = append(allErrs, field.Invalid(path.Child("schedule"), conf.Schedule,
			"PD Schedule Config is immutable through CRD, please modify with pd-ctl instead."))
	}
	if !reflect.DeepEqual(old.Replication, conf.Replication) {
		allErrs = append(allErrs, field.Invalid(path.Child("replication"), conf.Replication,
			"PD Replication Config is immutable through CRD, please modify with pd-ctl instead."))
	}
	return allErrs
}

func validateDeleteSlots(annotations map[string]string, key string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if annotations != nil {
		if value, ok := annotations[key]; ok {
			var slice []int32
			err := json.Unmarshal([]byte(value), &slice)
			if err != nil {
				msg := fmt.Sprintf("value of %q annotation must be a JSON list of int32", key)
				allErrs = append(allErrs, field.Invalid(fldPath, value, msg))
			}
		}
	}
	return allErrs
}
