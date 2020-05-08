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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	ClusterClientTLSPath = "/var/lib/cluster-client-tls"
	TiDBClientTLSPath    = "/var/lib/tidb-client-tls"
)

func GetOrdinalFromPodName(podName string) (int32, error) {
	ordinalStr := podName[strings.LastIndex(podName, "-")+1:]
	ordinalInt, err := strconv.Atoi(ordinalStr)
	if err != nil {
		return int32(0), err
	}
	return int32(ordinalInt), nil
}

func IsPodOrdinalNotExceedReplicas(pod *corev1.Pod, sts *appsv1.StatefulSet) (bool, error) {
	ordinal, err := GetOrdinalFromPodName(pod.Name)
	if err != nil {
		return false, err
	}
	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		return helper.GetPodOrdinals(*sts.Spec.Replicas, sts).Has(ordinal), nil
	}
	return ordinal < *sts.Spec.Replicas, nil
}

func getDeleteSlots(tc *v1alpha1.TidbCluster, annKey string) (deleteSlots sets.Int32) {
	deleteSlots = sets.NewInt32()
	annotations := tc.GetAnnotations()
	if annotations == nil {
		return
	}
	value, ok := annotations[annKey]
	if !ok {
		return
	}
	var slice []int32
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deleteSlots.Insert(slice...)
	return
}

// GetPodOrdinals gets desired ordials of member in given TidbCluster.
func GetPodOrdinals(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) (sets.Int32, error) {
	var ann string
	var replicas int32
	if memberType == v1alpha1.PDMemberType {
		ann = label.AnnPDDeleteSlots
		replicas = tc.Spec.PD.Replicas
	} else if memberType == v1alpha1.TiKVMemberType {
		ann = label.AnnTiKVDeleteSlots
		replicas = tc.Spec.TiKV.Replicas
	} else if memberType == v1alpha1.TiDBMemberType {
		ann = label.AnnTiDBDeleteSlots
		replicas = tc.Spec.TiDB.Replicas
	} else {
		return nil, fmt.Errorf("unknown member type %v", memberType)
	}
	deleteSlots := getDeleteSlots(tc, ann)
	maxReplicaCount, deleteSlots := helper.GetMaxReplicaCountAndDeleteSlots(replicas, deleteSlots)
	podOrdinals := sets.NewInt32()
	for i := int32(0); i < maxReplicaCount; i++ {
		if !deleteSlots.Has(i) {
			podOrdinals.Insert(i)
		}
	}
	return podOrdinals, nil
}

func OrdinalPVCName(memberType v1alpha1.MemberType, setName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", memberType, setName, ordinal)
}

// IsSubMapOf returns whether the first map is a sub map of the second map
func IsSubMapOf(first map[string]string, second map[string]string) bool {
	for k, v := range first {
		if second == nil {
			return false
		}
		if second[k] != v {
			return false
		}
	}
	return true
}

func GetPodName(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", tc.Name, memberType.String(), ordinal)
}

func IsStatefulSetUpgrading(set *appsv1.StatefulSet) bool {
	return !(set.Status.CurrentRevision == set.Status.UpdateRevision)
}

func IsStatefulSetScaling(set *appsv1.StatefulSet) bool {
	return !(set.Status.Replicas == *set.Spec.Replicas)
}

func GetStatefulSetName(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) string {
	return fmt.Sprintf("%s-%s", tc.Name, memberType.String())
}

func GetAutoScalingOutSlots(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) sets.Int32 {
	s := sets.Int32{}
	l := ""
	switch memberType {
	case v1alpha1.PDMemberType:
		return s
	case v1alpha1.TiKVMemberType:
		l = label.AnnTiKVAutoScalingOutOrdinals
	case v1alpha1.TiDBMemberType:
		l = label.AnnTiDBAutoScalingOutOrdinals
	default:
		return s
	}
	if tc.Annotations == nil {
		return s
	}
	v, existed := tc.Annotations[l]
	if !existed {
		return s
	}
	var slice []int32
	err := json.Unmarshal([]byte(v), &slice)
	if err != nil {
		return s
	}
	s.Insert(slice...)
	return s
}

func Encode(obj interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ClusterClientTLSSecretName(tcName string) string {
	return fmt.Sprintf("%s-cluster-client-secret", tcName)
}

func ClusterTLSSecretName(tcName, component string) string {
	return fmt.Sprintf("%s-%s-cluster-secret", tcName, component)
}

func TiDBClientTLSSecretName(tcName string) string {
	return fmt.Sprintf("%s-tidb-client-secret", tcName)
}

// SortEnvByName implements sort.Interface to sort env list by name.
type SortEnvByName []corev1.EnvVar

func (e SortEnvByName) Len() int {
	return len(e)
}
func (e SortEnvByName) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e SortEnvByName) Less(i, j int) bool {
	return e[i].Name < e[j].Name
}

// AppendEnv appends envs `b` into `a` ignoring envs whose names already exist
// in `b`.
// Note that this will not change relative order of envs.
func AppendEnv(a []corev1.EnvVar, b []corev1.EnvVar) []corev1.EnvVar {
	aMap := make(map[string]corev1.EnvVar)
	for _, e := range a {
		aMap[e.Name] = e
	}
	for _, e := range b {
		if _, ok := aMap[e.Name]; !ok {
			a = append(a, e)
		}
	}
	return a
}

// IsOwnedByTidbCluster checks if the given object is owned by TidbCluster.
// Schema Kind and Group are checked, Version is ignored.
func IsOwnedByTidbCluster(obj metav1.Object) (bool, *metav1.OwnerReference) {
	ref := metav1.GetControllerOf(obj)
	if ref == nil {
		return false, nil
	}
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return false, nil
	}
	return ref.Kind == v1alpha1.TiDBClusterKind && gv.Group == v1alpha1.SchemeGroupVersion.Group, ref
}

// RemainNodeport is to make service nodeport unchanged during each reconciliation
func RemainNodeport(desiredSvc, existedSvc *corev1.Service) {
	if desiredSvc.Spec.Type != corev1.ServiceTypeNodePort && desiredSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return
	}
	for id, dport := range desiredSvc.Spec.Ports {
		for _, eport := range existedSvc.Spec.Ports {
			if dport.Port == eport.Port && dport.Protocol == eport.Protocol {
				dport.NodePort = eport.NodePort
				desiredSvc.Spec.Ports[id] = dport
				break
			}
		}
	}
}
