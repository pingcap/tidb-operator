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

package reloadable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestConvertOverlay(t *testing.T) {
	cases := []struct {
		desc string
		in   *v1alpha1.Overlay
		out  *v1alpha1.Overlay
	}{
		{
			desc: "nil",
			in:   nil,
			out: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{},
				},
			},
		},
		{
			desc: "pod is nil",
			in:   &v1alpha1.Overlay{},
			out: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{},
				},
			},
		},
		{
			desc: "pod spec is nil",
			in: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{},
			},
			out: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{},
				},
			},
		},
		{
			desc: "ignore pod labels and annotations",
			in: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					ObjectMeta: v1alpha1.ObjectMeta{
						Labels: map[string]string{
							"test": "test",
						},
						Annotations: map[string]string{
							"test": "test",
						},
					},
				},
			},
			out: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{},
				},
			},
		},
		{
			desc: "ignore pvcs",
			in: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{},
				PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
					{
						Name:                  "aaa",
						PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{},
					},
				},
			},
			out: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{},
				},
			},
		},
		{
			desc: "ignore container image",
			in: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name:  "bbb",
								Image: "yyy",
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "aaa",
								Image: "yyy",
							},
						},
					},
				},
			},
			out: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "bbb",
							},
						},
						Containers: []corev1.Container{
							{
								Name: "aaa",
							},
						},
					},
				},
			},
		},
		{
			desc: "keep main contianer image",
			in: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "pd",
								Image: "yyy",
							},
							{
								Name:  "tikv",
								Image: "yyy",
							},
							{
								Name:  "tiflash",
								Image: "yyy",
							},
							{
								Name:  "ticdc",
								Image: "yyy",
							},
							{
								Name:  "tso",
								Image: "yyy",
							},
							{
								Name:  "scheduler",
								Image: "yyy",
							},
							{
								Name:  "tiproxy",
								Image: "yyy",
							},
							{
								Name:  "tidb",
								Image: "yyy",
							},
						},
					},
				},
			},
			out: &v1alpha1.Overlay{
				Pod: &v1alpha1.PodOverlay{
					Spec: &corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "pd",
								Image: "yyy",
							},
							{
								Name:  "tikv",
								Image: "yyy",
							},
							{
								Name:  "tiflash",
								Image: "yyy",
							},
							{
								Name:  "ticdc",
								Image: "yyy",
							},
							{
								Name:  "tso",
								Image: "yyy",
							},
							{
								Name:  "scheduler",
								Image: "yyy",
							},
							{
								Name:  "tiproxy",
								Image: "yyy",
							},
							{
								Name:  "tidb",
								Image: "yyy",
							},
						},
					},
				},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			res := convertOverlay(c.in)
			assert.Equal(tt, c.out, res, c.desc)
		})
	}
}

func TestConvertVolumes(t *testing.T) {
	cases := []struct {
		desc string
		in   []v1alpha1.Volume
		out  []v1alpha1.Volume
	}{
		{
			desc: "nil",
		},
		{
			desc: "empty",
			in:   []v1alpha1.Volume{},
			out:  []v1alpha1.Volume{},
		},
		{
			desc: "ignore storage and storage class and volume attribute class",
			in: []v1alpha1.Volume{
				{
					Name:                      "aaa",
					Storage:                   resource.MustParse("5Gi"),
					StorageClassName:          ptr.To("test"),
					VolumeAttributesClassName: ptr.To("test"),
				},
				{
					Name:                      "bbb",
					Storage:                   resource.MustParse("5Gi"),
					StorageClassName:          ptr.To("test"),
					VolumeAttributesClassName: ptr.To("test"),
				},
			},
			out: []v1alpha1.Volume{
				{
					Name: "aaa",
				},
				{
					Name: "bbb",
				},
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			res := convertVolumes(c.in)
			assert.Equal(tt, c.out, res, c.desc)
		})
	}
}

func TestConvertLabels(t *testing.T) {
	cases := []struct {
		desc string
		in   map[string]string
		out  map[string]string
	}{
		{
			desc: "nil",
		},
		{
			desc: "empty",
			in:   map[string]string{},
			out:  map[string]string{},
		},
		{
			desc: "ignore keys",
			in: map[string]string{
				v1alpha1.LabelKeyManagedBy:            "xxx",
				v1alpha1.LabelKeyComponent:            "yyy",
				v1alpha1.LabelKeyCluster:              "zzz",
				v1alpha1.LabelKeyGroup:                "aaa",
				v1alpha1.LabelKeyInstanceRevisionHash: "bbb",
				"test":                                "test",
			},
			out: map[string]string{
				"test": "test",
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			res := convertLabels(c.in)
			assert.Equal(tt, c.out, res, c.desc)
		})
	}
}

func TestConvertAnnotations(t *testing.T) {
	cases := []struct {
		desc string
		in   map[string]string
		out  map[string]string
	}{
		{
			desc: "nil",
		},
		{
			desc: "empty",
			in:   map[string]string{},
			out:  map[string]string{},
		},
		{
			desc: "ignore keys",
			in: map[string]string{
				v1alpha1.AnnoKeyInitialClusterNum: "10",
				v1alpha1.AnnoKeyDeferDelete:       "xxx",
				"test":                            "test",
			},
			out: map[string]string{
				"test": "test",
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			res := convertAnnotations(c.in)
			assert.Equal(tt, c.out, res, c.desc)
		})
	}
}
