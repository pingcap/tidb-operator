// Copyright 2019. PingCAP, Inc.
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

package v1alpha1

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog"
)

const (
	defaultTiDBImage   = "pingcap/tidb"
	defaultTiKVImage   = "pingcap/tikv"
	defaultPDImage     = "pingcap/pd"
	defaultBinlogImage = "pingcap/tidb-binlog"
)

// +k8s:deepcopy-gen=false
type TidbClusterStrategy struct{}

func (TidbClusterStrategy) NewObject() runtime.Object {
	return &TidbCluster{}
}

func (TidbClusterStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if tc, ok := castTidbCluster(obj); ok {
		setTidbClusterDefault(tc)
	}
}

func (TidbClusterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	// no op to not affect the cluster managed by old versions of the helm chart
}

func (TidbClusterStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	if tc, ok := castTidbCluster(obj); ok {
		return validateTidbCluster(tc)
	}
	return field.ErrorList{}
}

func (TidbClusterStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	oldTc, oldOk := castTidbCluster(old)
	tc, ok := castTidbCluster(obj)
	if ok && oldOk {
		return validateUpdateTidbCluster(oldTc, tc)
	}
	return field.ErrorList{}
}

func setTidbClusterDefault(tc *TidbCluster) {
	if tc.Spec.TiDB.BaseImage == "" {
		tc.Spec.TiDB.BaseImage = defaultTiDBImage
	}
	if tc.Spec.TiKV.BaseImage == "" {
		tc.Spec.TiKV.BaseImage = defaultTiKVImage
	}
	if tc.Spec.PD.BaseImage == "" {
		tc.Spec.PD.BaseImage = defaultPDImage
	}
	if tc.Spec.Pump != nil && tc.Spec.Pump.BaseImage == "" {
		tc.Spec.Pump.BaseImage = defaultBinlogImage
	}
	if tc.Spec.TiDB.Config == nil {
		tc.Spec.TiDB.Config = &TiDBConfig{}
	}
	if tc.Spec.TiKV.Config == nil {
		tc.Spec.TiKV.Config = &TiKVConfig{}
	}
	if tc.Spec.PD.Config == nil {
		tc.Spec.PD.Config = &PDConfig{}
	}
	if string(tc.Spec.ImagePullPolicy) == "" {
		tc.Spec.ImagePullPolicy = corev1.PullIfNotPresent
	}
}

func validateTidbCluster(tc *TidbCluster) field.ErrorList {
	allErrs := validateTidbClusterSpec(&tc.Spec, field.NewPath("spec"))
	return allErrs
}

func validateTidbClusterSpec(spec *TidbClusterSpec, path *field.Path) field.ErrorList {
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
	if spec.TiDB.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("tidb.config"), spec.TiDB.Config, "tidb.config must not be nil"))
	}
	if spec.TiKV.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.config"), spec.TiKV.Config, "tidb.config must not be nil"))
	}
	if spec.PD.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.config"), spec.PD.Config, "tidb.config must not be nil"))
	}
	return allErrs
}

func validateUpdateTidbCluster(old, tc *TidbCluster) field.ErrorList {

	// For now we disable the new cluster validation in update for backward compatibility for old versions of our helm chart
	// TODO(aylei): enable validation on new tidbcluster after we deprecated the old versions of helm chart officially
	// allErrs := validateTidbCluster(tc)

	allErrs := field.ErrorList{}
	if old.GetInstanceName() != tc.GetInstanceName() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("labels"), tc.Labels,
			"The instance must not be mutate or set value other than the cluster name"))
	}
	allErrs = append(allErrs, validateUpdatePDConfig(old.Spec.PD.Config, tc.Spec.PD.Config, field.NewPath("spec.pd.config"))...)
	allErrs = append(allErrs, disallowUsingLegacyAPIInNewCluster(old, tc)...)

	return allErrs
}

// disallowUsingLegacyAPIInNewCluster checks if user use the legacy API in newly create cluster during update
// TODO(aylei): this could be removed after we enable validateTidbCluster() in update, which is more strict
func disallowUsingLegacyAPIInNewCluster(old, tc *TidbCluster) field.ErrorList {
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

func validateUpdatePDConfig(old, conf *PDConfig, path *field.Path) field.ErrorList {
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

func castTidbCluster(obj runtime.Object) (*TidbCluster, bool) {
	tc, ok := obj.(*TidbCluster)
	if !ok {
		// impossible for non-malicious request, this usually indicates a client error when the strategy is used by webhook,
		// we simply ignore error requests
		klog.Errorf("Object %T is not v1alpah1.TidbCluster, cannot processed by TidbClusterStrategy", obj)
		return nil, false
	}
	return tc, true
}
