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
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateAnnotations(t *testing.T) {
	successCases := []struct {
		name string
		tc   v1alpha1.TidbCluster
	}{
		{
			name: "all-fields-valid",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnTiKVDeleteSlots: "[1,2]",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    &v1alpha1.PDConfig{},
					},
					TiKV: v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    &v1alpha1.TiKVConfig{},
					},
					TiDB: v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    &v1alpha1.TiDBConfig{},
					},
				},
			},
		},
		{
			name: "no delete slots",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: map[string]string{},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    &v1alpha1.PDConfig{},
					},
					TiKV: v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    &v1alpha1.TiKVConfig{},
					},
					TiDB: v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    &v1alpha1.TiDBConfig{},
					},
				},
			},
		},
		// TODO: more cases
	}

	for _, v := range successCases {
		if errs := validateAnnotations(v.tc.ObjectMeta.Annotations, field.NewPath("metadata", "annotations")); len(errs) != 0 {
			t.Errorf("[%s]: unexpected error: %v", v.name, errs)
		}
	}

	errorCases := []struct {
		name string
		tc   v1alpha1.TidbCluster
		errs []field.Error
	}{
		{
			name: "delete slots empty string",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnTiKVDeleteSlots: "",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    &v1alpha1.PDConfig{},
					},
					TiKV: v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    &v1alpha1.TiKVConfig{},
					},
					TiDB: v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    &v1alpha1.TiDBConfig{},
					},
				},
			},
			errs: []field.Error{
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "tikv.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
			},
		},
		{
			name: "delete slots invalid format",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "1,2,3",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					Version: "v3.0.8",
					PD: v1alpha1.PDSpec{
						BaseImage: "pingcap/pd",
						Config:    &v1alpha1.PDConfig{},
					},
					TiKV: v1alpha1.TiKVSpec{
						BaseImage: "pingcap/tikv",
						Config:    &v1alpha1.TiKVConfig{},
					},
					TiDB: v1alpha1.TiDBSpec{
						BaseImage: "pingcap/tidb",
						Config:    &v1alpha1.TiDBConfig{},
					},
				},
			},
			errs: []field.Error{
				{
					Type:   field.ErrorTypeInvalid,
					Detail: `value of "tidb.tidb.pingcap.com/delete-slots" annotation must be a JSON list of int32`,
				},
			},
		},
	}

	for _, v := range errorCases {
		errs := validateAnnotations(v.tc.ObjectMeta.Annotations, field.NewPath("metadata", "annotations"))
		if len(errs) != len(v.errs) {
			t.Errorf("[%s]: expected %d failures, got %d failures: %v", v.name, len(v.errs), len(errs), errs)
			continue
		}
		for i := range errs {
			if errs[i].Type != v.errs[i].Type {
				t.Errorf("[%s]: expected error type %q, got %q", v.name, v.errs[i].Type, errs[i].Type)
			}
			if !strings.Contains(errs[i].Detail, v.errs[i].Detail) {
				t.Errorf("[%s]: expected error errs[i].Detail %q, got %q", v.name, v.errs[i].Detail, errs[i].Detail)
			}
			if len(v.errs[i].Field) > 0 {
				if errs[i].Field != v.errs[i].Field {
					t.Errorf("[%s]: expected error field %q, got %q", v.name, v.errs[i].Field, errs[i].Field)
				}
			}
		}
	}
}

func TestValidateRequestsStorage(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                 string
		haveRequest          bool
		resourceRequirements corev1.ResourceRequirements
		expectedErrors       int
	}{
		{
			name:        "has request storage",
			haveRequest: true,
			resourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10G"),
				},
			},
			expectedErrors: 0,
		},
		{
			name:        "Empty request storage",
			haveRequest: false,
			resourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{},
			},
			expectedErrors: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newTidbCluster()
			if tt.haveRequest {
				tc.Spec.PD.ResourceRequirements = tt.resourceRequirements
				tc.Spec.TiKV.ResourceRequirements = tt.resourceRequirements
			}
			err := ValidateTidbCluster(tc)
			r := len(err)
			g.Expect(r).Should(Equal(tt.expectedErrors))
		})
	}
}

func newTidbCluster() *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{}
	tc.Name = "test-validate-requests-storage"
	tc.Namespace = "default"
	return tc
}
