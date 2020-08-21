// Copyright 2018 PingCAP, Inc.
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

package member

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStatefulSetIsUpgrading(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name            string
		update          func(*apps.StatefulSet)
		expectUpgrading bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		set := &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
		}
		if test.update != nil {
			test.update(set)
		}
		b := statefulSetIsUpgrading(set)
		if test.expectUpgrading {
			g.Expect(b).To(BeTrue())
		} else {
			g.Expect(b).To(BeFalse())
		}
	}
	tests := []*testcase{
		{
			name:            "ObservedGeneration is nil",
			update:          nil,
			expectUpgrading: false,
		},
		{
			name: "CurrentRevision not equal UpdateRevision",
			update: func(set *apps.StatefulSet) {
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v2"
			},
			expectUpgrading: true,
		},
		{
			name: "set.Generation > *set.Status.ObservedGeneration && *set.Spec.Replicas == set.Status.Replicas",
			update: func(set *apps.StatefulSet) {
				set.Generation = 1001
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v1"
				set.Status.Replicas = 3
				set.Spec.Replicas = func() *int32 { var i int32; i = 3; return &i }()
			},
			expectUpgrading: true,
		},
		{
			name: "replicas not equal",
			update: func(set *apps.StatefulSet) {
				set.Generation = 1001
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v1"
				set.Status.Replicas = 3
				set.Spec.Replicas = func() *int32 { var i int32; i = 2; return &i }()
			},
			expectUpgrading: false,
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}

func TestGetStsAnnotations(t *testing.T) {
	tests := []struct {
		name      string
		tc        *v1alpha1.TidbCluster
		component string
		expected  map[string]string
	}{
		{
			name: "nil",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			component: label.TiDBLabelVal,
			expected:  map[string]string{},
		},
		{
			name: "empty",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			component: label.TiDBLabelVal,
			expected:  map[string]string{},
		},
		{
			name: "tidb",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
			},
			component: label.TiDBLabelVal,
			expected: map[string]string{
				helper.DeleteSlotsAnn: "[1,2]",
			},
		},
		{
			name: "tidb but component is not tidb",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
			},
			component: label.PDLabelVal,
			expected:  map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStsAnnotations(tt.tc, tt.component)
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
		})
	}
}
