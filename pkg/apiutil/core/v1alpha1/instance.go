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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/overlay"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/k8s"
	maputil "github.com/pingcap/tidb-operator/pkg/utils/map"
)

func IsReady[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) bool {
	return scope.From[S](f).IsReady()
}

func IsAvailable[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F, minReadySeconds int64, now time.Time) bool {
	return scope.From[S](f).IsAvailable(minReadySeconds, now)
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

// RetryIfInstancesReadyButNotAvailable returns a retry duration
// If any instances are ready but not available, updater may do nothing and
// cannot watch more changes of instances.
// So always retry if any instances are ready but not available.
func RetryIfInstancesReadyButNotAvailable[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](ins []F, minReadySeconds int64) time.Duration {
	now := time.Now()
	for _, in := range ins {
		// ready but not available
		if !IsReady[S](in) || IsAvailable[S](in, minReadySeconds, now) {
			continue
		}

		cond := FindStatusCondition[S](in, v1alpha1.CondReady)
		d := now.Sub(cond.LastTransitionTime.Time)
		return d
	}

	return 0
}

func IsOffline[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) bool {
	return scope.From[S](f).IsOffline()
}

func Volumes[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) []v1alpha1.Volume {
	return scope.From[S](f).Volumes()
}

func PVCOverlay[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](f F) []v1alpha1.NamedPersistentVolumeClaimOverlay {
	return scope.From[S](f).PVCOverlay()
}

func PVCs[
	S scope.Instance[F, T],
	F client.Object,
	T runtime.Instance,
](cluster *v1alpha1.Cluster, obj F, ps ...PVCPatch) []*corev1.PersistentVolumeClaim {
	vols := Volumes[S](obj)
	var pvcs []*corev1.PersistentVolumeClaim
	nameToIndex := map[string]int{}
	for i := range vols {
		vol := &vols[i]
		nameToIndex[vol.Name] = i
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      persistentVolumeClaimName(PodName[S](obj), vol.Name),
				Namespace: obj.GetNamespace(),
				Labels:    PersistentVolumeClaimLabels[S](obj, vol.Name),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(obj, scope.GVK[S]()),
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: vol.Storage,
					},
				},
				StorageClassName: vol.StorageClassName,
			},
		}

		for _, p := range ps {
			p.Patch(vol, pvc)
		}

		pvcs = append(pvcs, pvc)
	}

	pvcOverlays := PVCOverlay[S](obj)

	for _, o := range pvcOverlays {
		index, ok := nameToIndex[o.Name]
		if !ok {
			// TODO(liubo02): it should be validated
			panic("vol name" + o.Name + "doesn't exist")
		}

		overlay.OverlayPersistentVolumeClaim(pvcs[index], &o.PersistentVolumeClaim)
	}

	return pvcs
}

type PVCPatch interface {
	Patch(vol *v1alpha1.Volume, pvc *corev1.PersistentVolumeClaim)
}

type PVCPatchFunc func(vol *v1alpha1.Volume, pvc *corev1.PersistentVolumeClaim)

func (f PVCPatchFunc) Patch(vol *v1alpha1.Volume, pvc *corev1.PersistentVolumeClaim) {
	f(vol, pvc)
}

func EnableVAC(enable bool) PVCPatch {
	return PVCPatchFunc(func(vol *v1alpha1.Volume, pvc *corev1.PersistentVolumeClaim) {
		if enable {
			pvc.Spec.VolumeAttributesClassName = vol.VolumeAttributesClassName
		}
	})
}

func WithLegacyK8sAppLabels() PVCPatch {
	return PVCPatchFunc(func(_ *v1alpha1.Volume, pvc *corev1.PersistentVolumeClaim) {
		pvc.Labels = maputil.MergeTo(pvc.Labels, k8s.LabelsK8sApp(
			pvc.Labels[v1alpha1.LabelKeyCluster],
			pvc.Labels[v1alpha1.LabelKeyComponent],
		))
	})
}
