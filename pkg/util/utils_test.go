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

package util

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetOrdinalFromPodName(t *testing.T) {
	g := NewGomegaWithT(t)

	i, err := GetOrdinalFromPodName("pod-1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(i).To(Equal(int32(1)))

	i, err = GetOrdinalFromPodName("pod-notint")
	g.Expect(err).To(HaveOccurred())
	g.Expect(i).To(Equal(int32(0)))
}

func TestIsSubMapOf(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(IsSubMapOf(
		nil,
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
		},
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{},
		map[string]string{
			"k1": "v1",
		})).To(BeTrue())
	g.Expect(IsSubMapOf(
		map[string]string{
			"k1": "v1",
			"k2": "v2",
		},
		map[string]string{
			"k1": "v1",
		})).To(BeFalse())
}

func TestGetPodOrdinals(t *testing.T) {
	tests := []struct {
		name        string
		tc          *v1alpha1.TidbCluster
		memberType  v1alpha1.MemberType
		deleteSlots sets.Int32
	}{
		{
			name: "no delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 3,
					},
				},
			},
			memberType:  v1alpha1.TiDBMemberType,
			deleteSlots: sets.NewInt32(0, 1, 2),
		},
		{
			name: "delete slots",
			tc: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						label.AnnTiDBDeleteSlots: "[1,2]",
					},
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiDB: &v1alpha1.TiDBSpec{
						Replicas: 3,
					},
				},
			},
			memberType:  v1alpha1.TiDBMemberType,
			deleteSlots: sets.NewInt32(0, 3, 4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPodOrdinals(tt.tc, tt.memberType)
			if err != nil {
				t.Error(err)
			}
			if !got.Equal(tt.deleteSlots) {
				t.Errorf("expects %v got %v", tt.deleteSlots.List(), got.List())
			}
		})
	}
}

func TestAppendEnv(t *testing.T) {
	tests := []struct {
		name string
		a    []corev1.EnvVar
		b    []corev1.EnvVar
		want []corev1.EnvVar
	}{
		{
			name: "envs whose names exist are ignored",
			a: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "xxx",
					Value: "xxx",
				},
			},
			b: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "barbar",
				},
				{
					Name:  "new",
					Value: "bar",
				},
				{
					Name:  "xxx",
					Value: "yyy",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
				{
					Name:  "xxx",
					Value: "xxx",
				},
				{
					Name:  "new",
					Value: "bar",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendEnv(tt.a, tt.b)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unwant (-want, +got): %s", diff)
			}
		})
	}
}
