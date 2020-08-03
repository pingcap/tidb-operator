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
	"fmt"
	"reflect"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestStrategyAdmissionHook_Admit(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name      string
		operation admissionv1beta1.Operation
		apiObj    runtime.Object

		expectedPrepareForCreateTimes int
		expectedPrepareForUpdateTimes int
	}
	testcases := []testcase{
		{
			name:                          "Prepare for create",
			operation:                     admissionv1beta1.Create,
			apiObj:                        &v1alpha1.TidbCluster{},
			expectedPrepareForCreateTimes: 1,
			expectedPrepareForUpdateTimes: 0,
		}, {
			name:                          "Prepare for update",
			operation:                     admissionv1beta1.Update,
			apiObj:                        &v1alpha1.TidbCluster{},
			expectedPrepareForCreateTimes: 0,
			expectedPrepareForUpdateTimes: 1,
		}, {
			name:                          "Deletion should be bypassed",
			operation:                     admissionv1beta1.Delete,
			apiObj:                        &v1alpha1.TidbCluster{},
			expectedPrepareForCreateTimes: 0,
			expectedPrepareForUpdateTimes: 0,
		}, {
			name:                          "Unregistered api type should be bypassed",
			operation:                     admissionv1beta1.Create,
			apiObj:                        &v1alpha1.Backup{},
			expectedPrepareForCreateTimes: 0,
			expectedPrepareForUpdateTimes: 0,
		},
	}

	testFn := func(tt *testcase) {
		t.Log(tt.name)
		r := NewRegistry()
		s := &FakeStrategy{}
		r.Register(s)
		w := NewStrategyAdmissionHook(&r)
		gvk, err := controller.InferObjectKind(tt.apiObj)
		g.Expect(err).To(Succeed())
		raw, err := json.Marshal(tt.apiObj)
		g.Expect(err).To(Succeed())

		re := runtime.RawExtension{
			Raw:    raw,
			Object: tt.apiObj,
		}
		ar := admissionv1beta1.AdmissionRequest{
			Kind: metav1.GroupVersionKind{
				Kind:    gvk.Kind,
				Group:   gvk.Group,
				Version: gvk.Version,
			},
			Operation: tt.operation,
			Object:    re,
		}
		if ar.Operation == admissionv1beta1.Update || ar.Operation == admissionv1beta1.Delete {
			ar.OldObject = *re.DeepCopy()
		}

		resp := w.Admit(&ar)
		g.Expect(resp.Allowed).To(BeTrue())
		g.Expect(s.prepareForCreateTracker.GetRequests()).To(Equal(tt.expectedPrepareForCreateTimes))
		g.Expect(s.prepareForUpdateTracker.GetRequests()).To(Equal(tt.expectedPrepareForUpdateTimes))
	}

	for _, tt := range testcases {
		testFn(&tt)
	}
}

func TestStrategyAdmissionHook_Validate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name      string
		operation admissionv1beta1.Operation
		apiObj    runtime.Object

		validateError       error
		validateUpdateError error

		expectedValidateTimes          int
		expectedValidateForUpdateTimes int
	}
	testcases := []testcase{
		{
			name:                           "Validate creating",
			operation:                      admissionv1beta1.Create,
			apiObj:                         &v1alpha1.TidbCluster{},
			expectedValidateTimes:          1,
			expectedValidateForUpdateTimes: 0,
		}, {
			name:                           "Validate updating",
			operation:                      admissionv1beta1.Update,
			apiObj:                         &v1alpha1.TidbCluster{},
			expectedValidateTimes:          0,
			expectedValidateForUpdateTimes: 1,
		}, {
			name:                           "Deletion should be bypassed",
			operation:                      admissionv1beta1.Delete,
			apiObj:                         &v1alpha1.TidbCluster{},
			expectedValidateTimes:          0,
			expectedValidateForUpdateTimes: 0,
		}, {
			name:                           "Unregistered api type should be bypassed",
			operation:                      admissionv1beta1.Create,
			apiObj:                         &v1alpha1.Backup{},
			expectedValidateTimes:          0,
			expectedValidateForUpdateTimes: 0,
		},
		{
			name:                           "Validate creating error",
			operation:                      admissionv1beta1.Create,
			apiObj:                         &v1alpha1.TidbCluster{},
			validateError:                  fmt.Errorf("invalid object"),
			expectedValidateTimes:          1,
			expectedValidateForUpdateTimes: 0,
		}, {
			name:                           "Validate updating error",
			operation:                      admissionv1beta1.Update,
			apiObj:                         &v1alpha1.TidbCluster{},
			validateUpdateError:            fmt.Errorf("invalid object"),
			expectedValidateTimes:          0,
			expectedValidateForUpdateTimes: 1,
		},
	}

	testFn := func(tt *testcase) {
		t.Log(tt.name)
		r := NewRegistry()
		s := &FakeStrategy{}
		r.Register(s)
		w := NewStrategyAdmissionHook(&r)
		gvk, err := controller.InferObjectKind(tt.apiObj)
		g.Expect(err).To(Succeed())
		raw, err := json.Marshal(tt.apiObj)
		g.Expect(err).To(Succeed())

		re := runtime.RawExtension{
			Raw:    raw,
			Object: tt.apiObj,
		}
		ar := admissionv1beta1.AdmissionRequest{
			Kind: metav1.GroupVersionKind{
				Kind:    gvk.Kind,
				Group:   gvk.Group,
				Version: gvk.Version,
			},
			Operation: tt.operation,
			Object:    re,
		}
		if ar.Operation == admissionv1beta1.Update || ar.Operation == admissionv1beta1.Delete {
			ar.OldObject = *re.DeepCopy()
		}

		if tt.validateError != nil {
			s.validateTracker.SetError(tt.validateError)
		}
		if tt.validateUpdateError != nil {
			s.validateUpdateTracker.SetError(tt.validateUpdateError)
		}

		resp := w.Validate(&ar)
		if tt.validateError != nil && tt.operation == admissionv1beta1.Create {
			g.Expect(resp.Allowed).To(BeFalse())
		} else if tt.validateUpdateError != nil && tt.operation == admissionv1beta1.Update {
			g.Expect(resp.Allowed).To(BeFalse())
		} else {
			g.Expect(resp.Allowed).To(BeTrue())
		}
		g.Expect(s.validateTracker.GetRequests()).To(Equal(tt.expectedValidateTimes))
		g.Expect(s.validateUpdateTracker.GetRequests()).To(Equal(tt.expectedValidateForUpdateTimes))
	}

	for _, tt := range testcases {
		testFn(&tt)
	}

}

type FakeStrategy struct {
	prepareForCreateTracker controller.RequestTracker
	prepareForUpdateTracker controller.RequestTracker
	validateTracker         controller.RequestTracker
	validateUpdateTracker   controller.RequestTracker
}

func (s *FakeStrategy) NewObject() runtime.Object {
	return &v1alpha1.TidbCluster{}
}

func (s *FakeStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	s.prepareForCreateTracker.Inc()
}

func (s *FakeStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	s.prepareForUpdateTracker.Inc()
}

func (s *FakeStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	var allErrs field.ErrorList
	s.validateTracker.Inc()
	if s.validateTracker.ErrorReady() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), "", s.validateTracker.GetError().Error()))
	}
	return allErrs
}

func (s *FakeStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	var allErrs field.ErrorList
	s.validateUpdateTracker.Inc()
	if s.validateUpdateTracker.ErrorReady() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), "", s.validateUpdateTracker.GetError().Error()))
	}
	return allErrs
}

func TestValidatingResource(t *testing.T) {
	r := NewRegistry()
	w := NewStrategyAdmissionHook(&r)
	wantGvr := schema.GroupVersionResource{
		Group:    "admission.tidb.pingcap.com",
		Version:  "v1alpha1",
		Resource: "pingcapresourcevalidations",
	}
	gvr, _ := w.ValidatingResource()
	if !reflect.DeepEqual(wantGvr, gvr) {
		t.Fatalf("want: %v, got: %v", wantGvr, gvr)
	}
}

func TestMutationResource(t *testing.T) {
	r := NewRegistry()
	w := NewStrategyAdmissionHook(&r)
	wantGvr := schema.GroupVersionResource{
		Group:    "admission.tidb.pingcap.com",
		Version:  "v1alpha1",
		Resource: "pingcapresourcemutations",
	}
	gvr, _ := w.MutatingResource()
	if !reflect.DeepEqual(wantGvr, gvr) {
		t.Fatalf("want: %v, got: %v", wantGvr, gvr)
	}
}
