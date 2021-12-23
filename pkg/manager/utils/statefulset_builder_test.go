// Copyright 2021 PingCAP, Inc.
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

package utils

import (
	"testing"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStatefulSetBuilder(t *testing.T) {
	stsTmpl := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "src-sts",
		},
		Spec: apps.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container",
						},
					},
				},
			},
		},
	}

	t.Run("Basic", func(t *testing.T) {
		srcSts := stsTmpl.DeepCopy()
		builder := NewStatefulSetBuilder(srcSts)

		testStatefulSetBuilderBasicFn(t, builder)
	})

	t.Run("Get", func(t *testing.T) {
		g := NewGomegaWithT(t)

		srcSts := stsTmpl.DeepCopy()
		builder := NewStatefulSetBuilder(srcSts)

		sts := builder.Get()
		g.Expect(sts == srcSts).Should(BeTrue())
	})

	t.Run("Clone", func(t *testing.T) {
		g := NewGomegaWithT(t)

		srcSts := stsTmpl.DeepCopy()
		builder := NewStatefulSetBuilder(srcSts)

		sts := builder.Clone()
		g.Expect(sts == srcSts).Should(BeFalse())
		g.Expect(sts).Should(Equal(srcSts))

		sts.Name = "sts"
		g.Expect(sts).ShouldNot(Equal(srcSts))
	})

	t.Run("PodTemplateSpecBuilder", func(t *testing.T) {
		g := NewGomegaWithT(t)

		srcSts := stsTmpl.DeepCopy()
		builder := NewStatefulSetBuilder(srcSts)

		podBuilder := builder.PodTemplateSpecBuilder()
		testPodTemplateSpecBuilderBasicFn(t, podBuilder)
		g.Expect(&builder.prototype.Spec.Template == podBuilder.prototype).Should(BeTrue())
	})
}

func TestPodTemplateSpecBuilder(t *testing.T) {
	tmpl := &corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container",
				},
			},
		},
	}

	t.Run("Basic", func(t *testing.T) {
		src := tmpl.DeepCopy()
		builder := NewPodTemplateSpecBuilder(src)

		testPodTemplateSpecBuilderBasicFn(t, builder)
	})

	t.Run("Get", func(t *testing.T) {
		g := NewGomegaWithT(t)

		src := tmpl.DeepCopy()
		builder := NewPodTemplateSpecBuilder(src)

		podtemplate := builder.Get()
		g.Expect(podtemplate == src).Should(BeTrue())
	})

	t.Run("Clone", func(t *testing.T) {
		g := NewGomegaWithT(t)

		src := tmpl.DeepCopy()
		builder := NewPodTemplateSpecBuilder(src)
		podtemplate := builder.Clone()
		g.Expect(podtemplate == src).Should(BeFalse())
		g.Expect(podtemplate).Should(Equal(src))

		podtemplate.Name = "podtemplate"
		g.Expect(podtemplate).ShouldNot(Equal(src))
	})

	t.Run("ContainerBuilder", func(t *testing.T) {
		g := NewGomegaWithT(t)

		src := tmpl.DeepCopy()
		builder := NewPodTemplateSpecBuilder(src)

		ctrBuilder := builder.ContainerBuilder("container")
		testContainerBuilderBasicFn(t, ctrBuilder)
		g.Expect(&builder.prototype.Spec.Containers[0] == ctrBuilder.prototype).Should(BeTrue())
	})
}

func TestContainerBuilderBuilder(t *testing.T) {
	tmpl := &corev1.Container{}

	t.Run("Basic", func(t *testing.T) {
		src := tmpl.DeepCopy()
		builder := NewContainerBuilder(src)

		testContainerBuilderBasicFn(t, builder)
	})

	t.Run("Get", func(t *testing.T) {
		g := NewGomegaWithT(t)

		src := tmpl.DeepCopy()
		builder := NewContainerBuilder(src)

		podtemplate := builder.Get()
		g.Expect(podtemplate == src).Should(BeTrue())
	})

	t.Run("Clone", func(t *testing.T) {
		g := NewGomegaWithT(t)

		src := tmpl.DeepCopy()
		builder := NewContainerBuilder(src)
		container := builder.Clone()
		g.Expect(container == src).Should(BeFalse())
		g.Expect(container).Should(Equal(src))

		container.Name = "container"
		g.Expect(container).ShouldNot(Equal(src))
	})
}

func testStatefulSetBuilderBasicFn(t *testing.T, builder *StatefulSetBuilder) {
	g := NewGomegaWithT(t)

	// AddVolumeClaims
	pvc1 := &corev1.PersistentVolumeClaim{}
	pvc1.Name = "pvc1"
	builder.prototype.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{*pvc1}
	pvc2 := pvc1.DeepCopy()
	pvc2.Name = "pvc2"
	builder.AddVolumeClaims(*pvc2)
	g.Expect(builder.prototype.Spec.VolumeClaimTemplates).Should(HaveLen(2))
	g.Expect(builder.prototype.Spec.VolumeClaimTemplates[0]).Should(Equal(*pvc1))
	g.Expect(builder.prototype.Spec.VolumeClaimTemplates[1]).Should(Equal(*pvc2))
}

func testPodTemplateSpecBuilderBasicFn(t *testing.T, builder *PodTemplateSpecBuilder) {
	g := NewGomegaWithT(t)

	// AddVolumes
	vol1 := &corev1.Volume{}
	vol1.Name = "vol1"
	builder.prototype.Spec.Volumes = []corev1.Volume{*vol1}
	vol2 := vol1.DeepCopy()
	vol2.Name = "vol2"
	builder.AddVolumes(*vol2)
	g.Expect(builder.prototype.Spec.Volumes).Should(HaveLen(2))
	g.Expect(builder.prototype.Spec.Volumes[0]).Should(Equal(*vol1))
	g.Expect(builder.prototype.Spec.Volumes[1]).Should(Equal(*vol2))

	// RunInHostNetwork
	builder.RunInHostNetwork()
	g.Expect(builder.prototype.Spec.DNSPolicy).Should(Equal(corev1.DNSClusterFirstWithHostNet))
}

func testContainerBuilderBasicFn(t *testing.T, builder *ContainerBuilder) {
	g := NewGomegaWithT(t)

	// AddVolumeMounts
	vol1 := &corev1.VolumeMount{}
	vol1.Name = "volumemount1"
	builder.prototype.VolumeMounts = []corev1.VolumeMount{*vol1}
	vol2 := vol1.DeepCopy()
	vol2.Name = "volumemount2"
	builder.AddVolumeMounts(*vol2)
	g.Expect(builder.prototype.VolumeMounts).Should(HaveLen(2))
	g.Expect(builder.prototype.VolumeMounts[0]).Should(Equal(*vol1))
	g.Expect(builder.prototype.VolumeMounts[1]).Should(Equal(*vol2))
}
