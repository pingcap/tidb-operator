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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

const (
	// LastAppliedConfigAnnotation is annotation key of last applied configuration
	LastAppliedConfigAnnotation = "pingcap.com/last-applied-configuration"
)

func timezoneMountVolume() (corev1.VolumeMount, corev1.Volume) {
	return corev1.VolumeMount{Name: "timezone", MountPath: "/etc/localtime", ReadOnly: true},
		corev1.Volume{
			Name:         "timezone",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/localtime"}},
		}
}

func annotationsMountVolume() (corev1.VolumeMount, corev1.Volume) {
	m := corev1.VolumeMount{Name: "annotations", ReadOnly: true, MountPath: "/etc/podinfo"}
	v := corev1.Volume{
		Name: "annotations",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "annotations",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			},
		},
	}
	return m, v
}

// statefulSetIsUpgrading confirms whether the statefulSet is upgrading phase
func statefulSetIsUpgrading(set *apps.StatefulSet) bool {
	if set.Status.CurrentRevision != set.Status.UpdateRevision {
		return true
	}
	if set.Generation > *set.Status.ObservedGeneration && *set.Spec.Replicas == set.Status.Replicas {
		return true
	}
	return false
}

// SetLastAppliedConfigAnnotation set last applied config info to Statefulset's annotation
func SetLastAppliedConfigAnnotation(set *apps.StatefulSet) error {
	setApply, err := encode(set.Spec)
	if err != nil {
		return err
	}
	if set.Annotations == nil {
		set.Annotations = map[string]string{}
	}
	set.Annotations[LastAppliedConfigAnnotation] = setApply

	templateApply, err := encode(set.Spec.Template.Spec)
	if err != nil {
		return err
	}
	if set.Spec.Template.Annotations == nil {
		set.Spec.Template.Annotations = map[string]string{}
	}
	set.Spec.Template.Annotations[LastAppliedConfigAnnotation] = templateApply
	return nil
}

// GetLastAppliedConfig get last applied config info from Statefulset's annotation
func GetLastAppliedConfig(set *apps.StatefulSet) (*apps.StatefulSetSpec, *corev1.PodSpec, error) {
	specAppliedConfig, ok := set.Annotations[LastAppliedConfigAnnotation]
	if !ok {
		return nil, nil, fmt.Errorf("statefulset:[%s/%s] not found spec's apply config", set.GetNamespace(), set.GetName())
	}
	spec := &apps.StatefulSetSpec{}
	err := json.Unmarshal([]byte(specAppliedConfig), spec)
	if err != nil {
		return nil, nil, err
	}

	podSpecAppliedConfig, ok := set.Spec.Template.Annotations[LastAppliedConfigAnnotation]
	if !ok {
		return nil, nil, fmt.Errorf("statefulset:[%s/%s] not found template spec's apply config", set.GetNamespace(), set.GetName())
	}
	podSpec := &corev1.PodSpec{}
	err = json.Unmarshal([]byte(podSpecAppliedConfig), podSpec)
	if err != nil {
		return nil, nil, err
	}

	return spec, podSpec, nil
}

func encode(obj interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// statefulSetEqual compares the new Statefulset's spec with old Statefulset's last applied config
func statefulSetEqual(new apps.StatefulSet, old apps.StatefulSet) bool {
	oldConfig := apps.StatefulSetSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldConfig)
		if err != nil {
			glog.Errorf("unmarshal Statefulset: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false
		}
		return apiequality.Semantic.DeepEqual(oldConfig, new.Spec)
	}
	return false
}

// templateEqual compares the new podTemplateSpec's spec with old podTemplateSpec's last applied config
func templateEqual(new corev1.PodTemplateSpec, old corev1.PodTemplateSpec) bool {
	oldConfig := corev1.PodSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldConfig)
		if err != nil {
			glog.Errorf("unmarshal PodTemplate: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false
		}
		return apiequality.Semantic.DeepEqual(oldConfig, new.Spec)
	}
	return false
}

// SetServiceLastAppliedConfigAnnotation set last applied config info to Service's annotation
func SetServiceLastAppliedConfigAnnotation(svc *corev1.Service) error {
	svcApply, err := encode(svc.Spec)
	if err != nil {
		return err
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[LastAppliedConfigAnnotation] = svcApply
	return nil
}

// serviceEqual compares the new Service's spec with old Service's last applied config
func serviceEqual(new, old *corev1.Service) (bool, error) {
	oldSpec := corev1.ServiceSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigAnnotation]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			glog.Errorf("unmarshal ServiceSpec: [%s/%s]'s applied config failed,error: %v", old.GetNamespace(), old.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldSpec, new.Spec), nil
	}
	return false, nil
}

// statefulPodRegex is a regular expression that extracts the parent StatefulSet and ordinal from the Name of a Pod
var statefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

func getOrdinal(pod *corev1.Pod) int {
	ordinal := -1
	subMatches := statefulPodRegex.FindStringSubmatch(pod.Name)
	if len(subMatches) < 3 {
		return ordinal
	}
	if i, err := strconv.ParseInt(subMatches[2], 10, 32); err == nil {
		ordinal = int(i)
	}
	return ordinal
}

// ascendingOrdinal is a sort.Interface that Sorts a list of Pods based on the ordinals extracted
// from the Pod. Pod's that have not been constructed by StatefulSet's have an ordinal of -1, and are therefore pushed
// to the front of the list.
type descendingOrdinal []*corev1.Pod

func (ao descendingOrdinal) Len() int {
	return len(ao)
}

func (ao descendingOrdinal) Swap(i, j int) {
	ao[i], ao[j] = ao[j], ao[i]
}

func (ao descendingOrdinal) Less(i, j int) bool {
	return getOrdinal(ao[i]) > getOrdinal(ao[j])
}
