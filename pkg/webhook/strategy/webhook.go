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

package strategy

import (
	"context"
	"encoding/json"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

// StrategyAdmissionHook is a admission webhook based on the registered strategies in the given registry
type StrategyAdmissionHook struct {
	registry *StrategyRegistry
}

var _ apiserver.ValidatingAdmissionHook = &StrategyAdmissionHook{}
var _ apiserver.MutatingAdmissionHook = &StrategyAdmissionHook{}

func NewStrategyAdmissionHook(registry *StrategyRegistry) *StrategyAdmissionHook {
	return &StrategyAdmissionHook{registry}
}

func (w *StrategyAdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "pingcapresourcevalidations",
		},
		"pingcapresourcevalidation"
}

func (w *StrategyAdmissionHook) MutatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.tidb.pingcap.com",
			Version:  "v1alpha1",
			Resource: "pingcapresourcemutations",
		},
		"pingcapresourcemutation"
}

func (w *StrategyAdmissionHook) Validate(ar *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	s, ok := w.registry.Get(ar.Kind)
	if !ok {
		// no strategy registered
		return util.ARSuccess()
	}
	if ar.Operation != admissionv1beta1.Create && ar.Operation != admissionv1beta1.Update {
		return util.ARSuccess()
	}
	obj := s.NewObject()
	if err := json.Unmarshal(ar.Object.Raw, obj); err != nil {
		klog.Errorf("admission validating failed: cannot unmarshal %s to %T", ar.Kind, obj)
		return util.ARFail(err)
	}
	var allErr field.ErrorList
	if ar.Operation == admissionv1beta1.Create {
		allErr = s.Validate(context.TODO(), obj)
	} else {
		old := s.NewObject()
		if err := json.Unmarshal(ar.OldObject.Raw, old); err != nil {
			klog.Errorf("admission validating failed: cannot unmarshal %s to %T", ar.Kind, old)
			return util.ARFail(err)
		}
		allErr = s.ValidateUpdate(context.TODO(), obj, old)
	}
	if len(allErr) > 0 {
		return util.ARFail(allErr.ToAggregate())
	}
	return util.ARSuccess()
}

func (w *StrategyAdmissionHook) Admit(ar *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	s, ok := w.registry.Get(ar.Kind)
	if !ok {
		return util.ARSuccess()
	}
	if ar.Operation != admissionv1beta1.Create && ar.Operation != admissionv1beta1.Update {
		return util.ARSuccess()
	}
	obj := s.NewObject()
	if err := json.Unmarshal(ar.Object.Raw, obj); err != nil {
		klog.Errorf("admission validating failed: cannot unmarshal %s to %T", ar.Kind, obj)
		return util.ARFail(err)
	}
	original := obj.DeepCopyObject()
	if ar.Operation == admissionv1beta1.Create {
		s.PrepareForCreate(context.TODO(), obj)
	} else {
		old := s.NewObject()
		if err := json.Unmarshal(ar.OldObject.Raw, old); err != nil {
			klog.Errorf("admission validating failed: cannot unmarshal %s to %T", ar.Kind, old)
			return util.ARFail(err)
		}
		s.PrepareForUpdate(context.TODO(), obj, old)
	}
	patch, err := util.CreateJsonPatch(original, obj)
	if err != nil {
		return util.ARFail(err)
	}
	return util.ARPatch(patch)
}

func (w *StrategyAdmissionHook) Initialize(cfg *rest.Config, stopCh <-chan struct{}) error {
	return nil
}
