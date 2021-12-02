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

package statefulset

import (
	"github.com/pingcap/tidb-operator/pkg/util"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type ContainerBuilder struct {
	prototype *corev1.Container
}

func NewContainerBuilder(container *corev1.Container) *ContainerBuilder {
	return &ContainerBuilder{
		prototype: container,
	}
}

func (cb *ContainerBuilder) Get() *corev1.Container {
	return cb.prototype
}

func (cb *ContainerBuilder) Clone() *corev1.Container {
	return cb.prototype.DeepCopy()
}

func (cb *ContainerBuilder) AddVolumeMounts(mounts ...corev1.VolumeMount) {
	cb.prototype.VolumeMounts = append(cb.prototype.VolumeMounts, mounts...)
}

func (cb *ContainerBuilder) AddEnvs(envs ...corev1.EnvVar) {
	cb.prototype.Env = util.AppendEnv(cb.prototype.Env, envs)
}

type PodTemplateSpecBuilder struct {
	prototype *corev1.PodTemplateSpec
}

func NewPodTemplateSpecBuilder(podTemplate *corev1.PodTemplateSpec) *PodTemplateSpecBuilder {
	return &PodTemplateSpecBuilder{
		prototype: podTemplate,
	}
}

func (pb *PodTemplateSpecBuilder) Get() *corev1.PodTemplateSpec {
	return pb.prototype
}

func (pb *PodTemplateSpecBuilder) Clone() *corev1.PodTemplateSpec {
	return pb.prototype.DeepCopy()
}

func (pb *PodTemplateSpecBuilder) ContainerBuilder(name string) *ContainerBuilder {
	for i := range pb.prototype.Spec.Containers {
		container := pb.prototype.Spec.Containers[i]
		if container.Name == name {
			return NewContainerBuilder(&container)
		}
	}
	return nil
}

func (b *PodTemplateSpecBuilder) AddVolumes(volumes ...corev1.Volume) {
	b.prototype.Spec.Volumes = append(b.prototype.Spec.Volumes, volumes...)
}

func (b *PodTemplateSpecBuilder) AddLabels(labels map[string]string) {
	b.prototype.Labels = util.CombineStringMap(b.prototype.Labels, labels)
}

func (b *PodTemplateSpecBuilder) AddAnnotations(annos map[string]string) {
	b.prototype.Labels = util.CombineStringMap(b.prototype.Annotations, annos)
}

func (b *PodTemplateSpecBuilder) RunInHostNetwork() {
	b.prototype.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
	for _, container := range b.prototype.Spec.Containers {
		container.Env = util.AppendEnv(container.Env, []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		})
	}
}

type StatefulSetBuilder struct {
	prototype *apps.StatefulSet
}

func NewStatefulSetBuilder(sts *apps.StatefulSet) *StatefulSetBuilder {
	return &StatefulSetBuilder{
		prototype: sts,
	}
}

func (sb *StatefulSetBuilder) Get() *apps.StatefulSet {
	return sb.prototype
}

func (sb *StatefulSetBuilder) Clone() *apps.StatefulSet {
	return sb.prototype.DeepCopy()
}

func (sb *StatefulSetBuilder) PodTemplateSpecBuilder() *PodTemplateSpecBuilder {
	return NewPodTemplateSpecBuilder(&sb.prototype.Spec.Template)
}

func (sb *StatefulSetBuilder) AddVolumeClaims(pvcs ...corev1.PersistentVolumeClaim) {
	sb.prototype.Spec.VolumeClaimTemplates = append(sb.prototype.Spec.VolumeClaimTemplates, pvcs...)
}
