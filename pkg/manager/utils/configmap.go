// Copyright 2020 PingCAP, Inc.
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
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

func AddConfigMapDigestSuffix(cm *corev1.ConfigMap) error {
	sum, err := Sha256Sum(cm.Data)
	if err != nil {
		return err
	}
	suffix := fmt.Sprintf("%x", sum)[0:7]
	cm.Name = fmt.Sprintf("%s-%s", cm.Name, suffix)
	return nil
}

func Sha256Sum(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

// FindConfigMapVolume returns the configmap which's name matches the predicate in a PodSpec, empty indicates not found
func FindConfigMapVolume(podSpec *corev1.PodSpec, pred func(string) bool) string {
	for _, vol := range podSpec.Volumes {
		if vol.ConfigMap != nil && pred(vol.ConfigMap.LocalObjectReference.Name) {
			return vol.ConfigMap.LocalObjectReference.Name
		}
	}
	return ""
}

// FindConfigMapNameFromTCAnno is used to find ConfigMap name from tc.annotations which is saved before deleting STS.
// If the data of ConfigMap referenced in anno matches the newCm's data return the name in anno otherwise newCm's name.
//
// In some cases, we may need to delete and recreate STS for updating some immutable fields and are
// expected to keep the name of ConfigMap unchanged to ensure no accidentally restart of pod.
// For example: Updating storage size, iops or throughput of PVC using by TiKV. Now,
// the annotation is set by pvc_resizer(not supported yet), pvc_modifier or pvc_replacer, See pkg/manager/utils/statefulset.go:DeleteStatefulSetWithOrphan.
func FindConfigMapNameFromTCAnno(ctx context.Context, cmLister corelisters.ConfigMapLister, tc *v1alpha1.TidbCluster, componentType v1alpha1.MemberType, newCm *corev1.ConfigMap) (cmName string, _ error) {
	logger := klog.FromContext(ctx).WithValues("comp", componentType, "tc", fmt.Sprintf("%s/%s", tc.Namespace, tc.Name))
	cmNameInAnno := tc.Annotations[label.AnnoKeyOfConfigMapNameForNewSTS(string(componentType))]
	if cmNameInAnno == "" || cmNameInAnno == newCm.Name {
		return cmNameInAnno, nil
	}

	logger.Info("another cm name found in AnnoPrefixConfigMapNameBeforeDelete, try to use it as inuse name.", "cmName", newCm.Name, "nameInAnno", cmNameInAnno)
	cmInAnno, err := cmLister.ConfigMaps(tc.Namespace).Get(cmNameInAnno)
	if err != nil {
		return "", fmt.Errorf("failed to get configmap %s/%s: %w", tc.Namespace, cmNameInAnno, err)
	}
	// In some cases, ConfigMap may be changed between deleting and creating STS. For example: suspend a cluster and then
	// update its config. So just ignore the name in anno if ConfigMap data mismatches.
	if !equality.Semantic.DeepEqual(cmInAnno.Data, newCm.Data) {
		logger.Info("ConfigMap data changed, ignore the old name in Anno.", "cmName", newCm.Name, "cmNameInAnno", cmNameInAnno)
		return newCm.Name, nil
	}
	return cmNameInAnno, nil
}
