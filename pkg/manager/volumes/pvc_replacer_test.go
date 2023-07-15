// Copyright 2023 PingCAP, Inc.
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

package volumes

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

const (
	testMatchInterval = time.Millisecond * 100
	testMatchTimeout  = time.Second * 2
)

// STS = {replicas, claims{name, sz, class}...}
// Pods = {{name, sz, class}...}
type testVolDef struct {
	name string
	size string
	sc   string
}
type testSts struct {
	replicas int
	vols     []testVolDef
}
type testPod struct {
	vols []testVolDef
}

func makeTcAndK8Objects(deps *controller.Dependencies, g *GomegaWithT, sts testSts, pods []testPod) *v1alpha1.TidbCluster {
	ctx := context.Background()
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD:   &v1alpha1.PDSpec{},
			TiDB: &v1alpha1.TiDBSpec{},
			TiKV: &v1alpha1.TiKVSpec{
				Replicas: 3,
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": resource.MustParse("1Gi"),
					},
				},
				StorageClassName: pointer.StringPtr("storageclass-1"),
			},
		},
	}
	storageClasses := map[string]bool{"storageclass-1": true}
	makePVC := func(vol testVolDef) corev1.PersistentVolumeClaim {
		storageClasses[vol.sc] = true
		storageRequest := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(vol.size),
			},
		}
		return util.VolumeClaimTemplate(storageRequest, vol.name, pointer.StringPtr(vol.sc))
	}
	tikvSts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-tikv",
			Namespace: tc.Namespace,
		},
		Spec: apps.StatefulSetSpec{
			Replicas:             pointer.Int32Ptr(int32(sts.replicas)),
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{},
		},
	}
	for _, vol := range sts.vols {
		tikvSts.Spec.VolumeClaimTemplates = append(tikvSts.Spec.VolumeClaimTemplates, makePVC(vol))
	}
	tikvSts, err := deps.KubeClientset.AppsV1().StatefulSets(tc.Namespace).Create(ctx, tikvSts, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	podNames := []string{}
	pvcNames := []string{}
	for i, pod := range pods {
		tikvPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-cluster-tikv-%d", i),
				Namespace: tc.Namespace,
				Labels: map[string]string{
					label.InstanceLabelKey:  "test-cluster",
					label.ComponentLabelKey: label.TiKVLabelVal,
					label.ManagedByLabelKey: label.TiDBOperator,
					label.NameLabelKey:      "tidb-cluster",
				},
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{},
			},
		}
		for _, vol := range pod.vols {
			pvcName := fmt.Sprintf("%s-%s", vol.name, tikvPod.Name)
			tikvPod.Spec.Volumes = append(tikvPod.Spec.Volumes, corev1.Volume{
				Name: vol.name,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			})
			vol.name = pvcName
			pvc := makePVC(vol)
			_, err = deps.KubeClientset.CoreV1().PersistentVolumeClaims(tc.Namespace).Create(ctx, &pvc, metav1.CreateOptions{})
			pvcNames = append(pvcNames, pvc.Name)
			g.Expect(err).NotTo(HaveOccurred())
		}
		tikvPod, err = deps.KubeClientset.CoreV1().Pods(tikvPod.Namespace).Create(ctx, tikvPod, metav1.CreateOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		podNames = append(podNames, tikvPod.Name)
	}
	for sc := range storageClasses {
		_, err = deps.KubeClientset.StorageV1().StorageClasses().Create(ctx, newStorageClass(sc, false), metav1.CreateOptions{})
		g.Expect(err).NotTo(HaveOccurred())
	}
	// Wat till all objects available
	g.Eventually(func() error {
		_, err := deps.StatefulSetLister.StatefulSets(tc.Namespace).Get(tikvSts.Name)
		return err
	}, testMatchTimeout, testMatchInterval).Should(Succeed())
	for _, podName := range podNames {
		g.Eventually(func() error {
			_, err := deps.PodLister.Pods(tc.Namespace).Get(podName)
			return err
		}, testMatchTimeout, testMatchInterval).Should(Succeed())
	}
	for _, pvcName := range pvcNames {
		g.Eventually(func() error {
			_, err := deps.PVCLister.PersistentVolumeClaims(tc.Namespace).Get(pvcName)
			return err
		}, testMatchTimeout, testMatchInterval).Should(Succeed())
	}
	for sc := range storageClasses {
		g.Eventually(func() error {
			_, err := deps.StorageClassLister.Get(sc)
			return err
		}, testMatchTimeout, testMatchInterval).Should(Succeed())
	}
	return tc
}

func TestPvcReplacerStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name         string
		sts          testSts
		pods         []testPod
		expectStatus bool
	}
	testFn := func(tt testcase, t *testing.T) {
		deps := controller.NewFakeDependencies()
		stop := make(chan struct{})
		deps.KubeInformerFactory.Start(stop)
		deps.KubeInformerFactory.WaitForCacheSync(stop)
		defer close(stop)
		replacer := NewPVCReplacer(deps)
		tc := makeTcAndK8Objects(deps, g, tt.sts, tt.pods)
		replacer.UpdateStatus(tc)
		g.Expect(tc.Status.TiKV.VolReplaceInProgress).To(Equal(tt.expectStatus))
	}
	tests := []testcase{
		{
			name: "No Replace",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: false,
		},
		{
			name: "Replace due to Sts Storage Class",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-2"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
		{
			name: "Replace due to Sts Size",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "2Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
		{
			name: "Replace due to sts extra vol",
			sts: testSts{replicas: 3, vols: []testVolDef{
				{"tikv", "1Gi", "storageclass-1"},
				{"raftvol", "1Gi", "storageclass-1"},
			},
			},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
		{
			name: "Replace due to sts missing vol",
			sts:  testSts{replicas: 3, vols: []testVolDef{}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
		{
			name: "Replace due to pod Storage Class",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-2"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
		{
			name: "Replace due to pod size",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "2Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
		{
			name: "Replace due to pod extra vol",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{
					{"tikv", "1Gi", "storageclass-1"},
					{"raftvol", "1Gi", "storageclass-1"},
				}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
		{
			name: "Replace due to pod missing vol",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStatus: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}
func TestPvcReplacerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                   string
		sts                    testSts
		pods                   []testPod
		expectStsDeleted       bool
		expectAnnotationOnPod1 bool
		isScale                bool
	}
	testFn := func(tt testcase, t *testing.T) {
		deps := controller.NewFakeDependencies()
		stop := make(chan struct{})
		deps.KubeInformerFactory.Start(stop)
		deps.KubeInformerFactory.WaitForCacheSync(stop)
		defer close(stop)
		replacer := NewPVCReplacer(deps)
		tc := makeTcAndK8Objects(deps, g, tt.sts, tt.pods)
		if tt.isScale {
			tc.Status.TiKV.Phase = v1alpha1.ScalePhase
		}
		syncErr := replacer.Sync(tc)
		deps.KubeInformerFactory.WaitForCacheSync(stop)
		if tt.expectStsDeleted {
			g.Eventually(func() error {
				_, err := deps.StatefulSetLister.StatefulSets(tc.Namespace).Get("test-cluster-tikv")
				return err
			}, testMatchTimeout, testMatchInterval).Should(HaveOccurred())
			g.Expect(syncErr.Error()).To(ContainSubstring("recreate statefulset"))
		} else {
			g.Expect(syncErr.Error()).To(Not(ContainSubstring("recreate statefulset")))
		}
		if tt.expectAnnotationOnPod1 {
			g.Eventually(func() map[string]string {
				pod1, _ := deps.PodLister.Pods(tc.Namespace).Get("test-cluster-tikv-1")
				return pod1.Annotations
			}, testMatchTimeout, testMatchInterval).Should(HaveKeyWithValue(v1alpha1.ReplaceDiskAnnKey, v1alpha1.ReplaceDiskValueTrue))
			g.Expect(syncErr.Error()).To(ContainSubstring("started disk replace"))
		} else {
			g.Expect(syncErr.Error()).To(Not(ContainSubstring("started disk replace")))
		}
	}
	tests := []testcase{
		{
			name: "Nothing to Sync",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStsDeleted:       false,
			expectAnnotationOnPod1: false,
		},
		{
			name: "Delete Sts",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-2"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStsDeleted:       true,
			expectAnnotationOnPod1: false,
		},
		{
			name: "Pod annotated for delete",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-2"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStsDeleted:       false,
			expectAnnotationOnPod1: true,
		},
		{
			name: "No Updates when Scaling",
			sts:  testSts{replicas: 3, vols: []testVolDef{{"tikv", "1Gi", "storageclass-2"}}},
			pods: []testPod{
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-2"}}},
				{vols: []testVolDef{{"tikv", "1Gi", "storageclass-1"}}},
			},
			expectStsDeleted:       false,
			expectAnnotationOnPod1: false,
			isScale:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}
