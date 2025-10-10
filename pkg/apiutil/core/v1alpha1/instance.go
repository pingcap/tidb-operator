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

package coreutil

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
)

func IsReady[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) bool {
	return scope.From[S](f).IsReady()
}

func NamePrefixAndSuffix[
	F client.Object,
](f F) (prefix, suffix string) {
	name := f.GetName()
	return runtime.NamePrefixAndSuffix(name)
}

// PodName returns the default managed pod name of an instance
// TODO(liubo02): rename to more reasonable one
// TODO(liubo02): move to namer
func PodName[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) string {
	prefix, suffix := NamePrefixAndSuffix(f)
	return prefix + "-" + scope.Component[S]() + "-" + suffix
}

func Subdomain[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) string {
	return scope.From[S](f).Subdomain()
}

func ClusterTLSVolume[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) *corev1.Volume {
	certKeyPair := ClusterCertKeyPairSecretName[S](f)
	ca := ClusterCASecretName[S](f)

	if ca == certKeyPair {
		return &corev1.Volume{
			Name: v1alpha1.VolumeNameClusterTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: ca,
				},
			},
		}
	}

	return &corev1.Volume{
		Name: v1alpha1.VolumeNameClusterTLS,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: ca,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  corev1.ServiceAccountRootCAKey,
									Path: corev1.ServiceAccountRootCAKey,
								},
							},
						},
					},
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: certKeyPair,
							},
							// avoid mounting injected ca.crt
							Items: []corev1.KeyToPath{
								{
									Key:  corev1.TLSCertKey,
									Path: corev1.TLSCertKey,
								},
								{
									Key:  corev1.TLSPrivateKeyKey,
									Path: corev1.TLSPrivateKeyKey,
								},
							},
						},
					},
				},
			},
		},
	}
}

func UpdateRevision[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) string {
	return scope.From[S](f).GetUpdateRevision()
}

func CurrentRevision[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) string {
	return scope.From[S](f).CurrentRevision()
}

func instanceSubresourceLabels[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) map[string]string {
	obj := scope.From[S](f)

	return maputil.MergeTo(maputil.Select(obj.GetLabels(),
		v1alpha1.LabelKeyGroup,
		v1alpha1.LabelKeyInstanceRevisionHash,
	), map[string]string{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyComponent: obj.Component(),
		v1alpha1.LabelKeyCluster:   obj.Cluster(),
		v1alpha1.LabelKeyInstance:  f.GetName(),
	})
}

func PodLabels[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) map[string]string {
	return instanceSubresourceLabels[S](f)
}

func ConfigMapLabels[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) map[string]string {
	return instanceSubresourceLabels[S](f)
}

func PersistentVolumeClaimLabels[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F, volName string) map[string]string {
	return maputil.MergeTo(instanceSubresourceLabels[S](f), map[string]string{
		v1alpha1.LabelKeyVolumeName: volName,
	})
}

func OwnerGroup[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) *metav1.OwnerReference {
	owner := metav1.GetControllerOfNoCopy(f)
	gvk := scope.GVK[S]()
	if owner.APIVersion != gvk.GroupVersion().String() || owner.Kind != gvk.Kind+"Group" {
		return nil
	}
	return owner
}

func IsOffline[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) bool {
	return scope.From[S](f).IsOffline()
}
