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

package framework

import (
	"github.com/onsi/ginkgo"
	"github.com/pingcap/tidb-operator/pkg/label"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

func NewDefaultFramework(baseName string) *framework.Framework {
	f := framework.NewDefaultFramework(baseName)
	var c kubernetes.Interface
	ginkgo.BeforeEach(func() {
		c = f.ClientSet
	})
	ginkgo.AfterEach(func() {
		// tidb-operator may set persistentVolumeReclaimPolicy to Retain if
		// users reqeust this. To reduce storage usage, we try to clean them if
		// namespace is deleted.
		pvList, err := c.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
		if err != nil {
			framework.Logf("failed to list pvs: %v", err)
			return
		}
		var (
			total          int = len(pvList.Items)
			retainReleased int
			skipped        int
			failed         int
			succeeded      int
		)
		defer func() {
			framework.Logf("recycling orphan PVs (total: %d, retainReleased: %d, skipped: %d, failed: %d, succeeded: %d)", total, retainReleased, skipped, failed, succeeded)
		}()
		for _, pv := range pvList.Items {
			if pv.Spec.PersistentVolumeReclaimPolicy != v1.PersistentVolumeReclaimRetain || pv.Status.Phase != v1.VolumeReleased {
				continue
			}
			retainReleased++
			pvcNamespaceName, ok := pv.Labels[label.NamespaceLabelKey]
			if !ok {
				framework.Logf("label %q does not exist in PV %q", label.NamespaceLabelKey, pv.Name)
				failed++
				continue
			}
			_, err := c.CoreV1().Namespaces().Get(pvcNamespaceName, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				framework.Logf("failed to get namespace %q: %v", pvcNamespaceName, err)
				failed++
				continue
			}
			if apierrors.IsNotFound(err) {
				skipped++
				continue
			}
			pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimDelete
			_, err = c.CoreV1().PersistentVolumes().Update(&pv)
			if err != nil {
				failed++
				framework.Logf("failed to set PersistentVolumeReclaimPolicy of PV %q to Delete: %v", pv.Name, err)
			} else {
				succeeded++
				framework.Logf("successfully set PersistentVolumeReclaimPolicy of PV %q to Delete", pv.Name)
			}
		}
		return
	})
	return f
}
