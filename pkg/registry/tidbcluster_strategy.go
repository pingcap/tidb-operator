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

package registry

import (
	"context"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/defaulting"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog"
)

// +k8s:deepcopy-gen=false
type TidbClusterStrategy struct{}

func (TidbClusterStrategy) NewObject() runtime.Object {
	return &v1alpha1.TidbCluster{}
}

func (TidbClusterStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if tc, ok := castTidbCluster(obj); ok {
		defaulting.SetTidbClusterDefault(tc)
	}
}

func (TidbClusterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	// no op to not affect the cluster managed by old versions of the helm chart
}

func (TidbClusterStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	if tc, ok := castTidbCluster(obj); ok {
		return validation.ValidateCreateTidbCluster(tc)
	}
	return field.ErrorList{}
}

func (TidbClusterStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	oldTc, oldOk := castTidbCluster(old)
	tc, ok := castTidbCluster(obj)
	if ok && oldOk {
		return validation.ValidateUpdateTidbCluster(oldTc, tc)
	}
	return field.ErrorList{}
}

func castTidbCluster(obj runtime.Object) (*v1alpha1.TidbCluster, bool) {
	tc, ok := obj.(*v1alpha1.TidbCluster)
	if !ok {
		// impossible for non-malicious request, this usually indicates a client error when the strategy is used by webhook,
		// we simply ignore error requests
		klog.Errorf("Object %T is not v1alpah1.TidbCluster, cannot processed by TidbClusterStrategy", obj)
		return nil, false
	}
	return tc, true
}
