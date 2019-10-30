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

package meta

import (
	"testing"

	"fmt"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestMetaManagerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name             string
		podHasLabels     bool
		pvcHasLabels     bool
		pvcHasVolumeName bool
		podRefPvc        bool
		podUpdateErr     bool
		getClusterErr    bool
		getMemberErr     bool
		getStoreErr      bool
		pvcUpdateErr     bool
		pvUpdateErr      bool
		podChanged       bool
		pvcChanged       bool
		pvChanged        bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForMeta()
		ns := tc.GetNamespace()
		pv1 := newPV()
		pvc1 := newPVC(tc)
		pod1 := newPod(tc)

		if !test.podHasLabels {
			pod1.Labels = nil
		}
		if !test.pvcHasLabels {
			pvc1.Labels = nil
		}
		if !test.pvcHasVolumeName {
			pvc1.Spec.VolumeName = ""
		}

		if !test.podRefPvc {
			pod1.Spec = newPodSpec(v1alpha1.TiDBMemberType.String(), pvc1.GetName())
		}

		nmm, fakePodControl, fakePVCControl, fakePVControl, podIndexer, pvcIndexer, pvIndexer := newFakeMetaManager()
		err := podIndexer.Add(pod1)
		g.Expect(err).NotTo(HaveOccurred())
		err = pvcIndexer.Add(pvc1)
		g.Expect(err).NotTo(HaveOccurred())
		err = pvIndexer.Add(pv1)
		g.Expect(err).NotTo(HaveOccurred())

		if test.podUpdateErr {
			fakePodControl.SetUpdatePodError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.getClusterErr {
			fakePodControl.SetGetClusterError(errors.NewInternalError(fmt.Errorf("PD server failed")), 0)
		}
		if test.getMemberErr {
			fakePodControl.SetGetMemberError(errors.NewInternalError(fmt.Errorf("PD server failed")), 0)
		}
		if test.getStoreErr {
			fakePodControl.SetGetStoreError(errors.NewInternalError(fmt.Errorf("PD server failed")), 0)
		}
		if test.pvcUpdateErr {
			fakePVCControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.pvUpdateErr {
			fakePVControl.SetUpdatePVError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err = nmm.Sync(tc)
		if test.podUpdateErr || test.getClusterErr || test.getMemberErr || test.getStoreErr {
			g.Expect(err).To(HaveOccurred())

			pod, err := nmm.podLister.Pods(ns).Get(pod1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(podMetaInfoMatchDesire(pod)).To(Equal(false))

			pvc, err := nmm.pvcLister.PersistentVolumeClaims(ns).Get(pvc1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvcMetaInfoMatchDesire(pvc)).To(Equal(false))

			pv, err := nmm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvMetaInfoMatchDesire(ns, pv)).To(Equal(false))
		}

		if test.pvcUpdateErr {
			g.Expect(err).To(HaveOccurred())

			pvc, err := nmm.pvcLister.PersistentVolumeClaims(ns).Get(pvc1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvcMetaInfoMatchDesire(pvc)).To(Equal(false))

			pv, err := nmm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvMetaInfoMatchDesire(ns, pv)).To(Equal(false))
		}

		if test.pvUpdateErr {
			g.Expect(err).To(HaveOccurred())

			pv, err := nmm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvMetaInfoMatchDesire(ns, pv)).To(Equal(false))
		}

		if test.podChanged {
			pod, err := nmm.podLister.Pods(ns).Get(pod1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(podMetaInfoMatchDesire(pod)).To(Equal(true))
		} else {
			pod, err := nmm.podLister.Pods(ns).Get(pod1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(podMetaInfoMatchDesire(pod)).To(Equal(false))
		}

		if test.pvcChanged {
			pvc, err := nmm.pvcLister.PersistentVolumeClaims(ns).Get(pvc1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvcMetaInfoMatchDesire(pvc)).To(Equal(true))
		} else {
			pvc, err := nmm.pvcLister.PersistentVolumeClaims(ns).Get(pvc1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvcMetaInfoMatchDesire(pvc)).To(Equal(false))
		}

		if test.pvChanged {
			g.Expect(err).NotTo(HaveOccurred())
			pv, err := nmm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvMetaInfoMatchDesire(ns, pv)).To(Equal(true))
		} else {
			pv, err := nmm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pvMetaInfoMatchDesire(ns, pv)).To(Equal(false))
		}
	}

	tests := []testcase{
		{
			name:             "normal",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       true,
			pvcChanged:       true,
			pvChanged:        true,
		},
		{
			name:             "pod don't have labels",
			podHasLabels:     false,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       false,
			pvcChanged:       false,
			pvChanged:        false,
		},
		{
			name:             "pvc don't have labels",
			podHasLabels:     true,
			pvcHasLabels:     false,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       true,
			pvcChanged:       true,
			pvChanged:        true,
		},
		{
			name:             "pvc don't have volumeName",
			podHasLabels:     true,
			pvcHasLabels:     false,
			pvcHasVolumeName: false,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       true,
			pvcChanged:       true,
			pvChanged:        false,
		},
		{
			name:             "the volume name of the pod is not tikv and pd",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        false,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       true,
			pvcChanged:       false,
			pvChanged:        false,
		},
		{
			name:             "pod update failed",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     true,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       false,
			pvcChanged:       false,
			pvChanged:        false,
		},
		{
			name:             "get cluster ID from PD failed",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    true,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       false,
			pvcChanged:       false,
			pvChanged:        false,
		},
		{
			name:             "get member ID from PD failed",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     true,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       false,
			pvcChanged:       false,
			pvChanged:        false,
		},
		{
			name:             "get store ID from PD failed",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      true,
			pvcUpdateErr:     false,
			pvUpdateErr:      false,
			podChanged:       false,
			pvcChanged:       false,
			pvChanged:        false,
		},
		{
			name:             "pvc update failed",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     true,
			pvUpdateErr:      false,
			podChanged:       true,
			pvcChanged:       false,
			pvChanged:        false,
		},
		{
			name:             "pv update failed",
			podHasLabels:     true,
			pvcHasLabels:     true,
			pvcHasVolumeName: true,
			podRefPvc:        true,
			podUpdateErr:     false,
			getClusterErr:    false,
			getMemberErr:     false,
			getStoreErr:      false,
			pvcUpdateErr:     false,
			pvUpdateErr:      true,
			podChanged:       true,
			pvcChanged:       true,
			pvChanged:        false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeMetaManager() (
	*metaManager,
	*controller.FakePodControl,
	*controller.FakePVCControl,
	*controller.FakePVControl,
	cache.Indexer,
	cache.Indexer,
	cache.Indexer,
) {
	kubeCli := kubefake.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	podInformer := kubeInformerFactory.Core().V1().Pods()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()

	podControl := controller.NewFakePodControl(podInformer)
	pvcControl := controller.NewFakePVCControl(pvcInformer)
	pvControl := controller.NewFakePVControl(pvInformer, pvcInformer)

	return &metaManager{
		pvcInformer.Lister(),
		pvcControl,
		pvInformer.Lister(),
		pvControl,
		podInformer.Lister(),
		podControl,
	}, podControl, pvcControl, pvControl, podInformer.Informer().GetIndexer(), pvcInformer.Informer().GetIndexer(), pvInformer.Informer().GetIndexer()
}

func podMetaInfoMatchDesire(pod *corev1.Pod) bool {
	return pod.Labels[label.ClusterIDLabelKey] == controller.TestClusterID &&
		pod.Labels[label.MemberIDLabelKey] == controller.TestMemberID &&
		pod.Labels[label.StoreIDLabelKey] == controller.TestStoreID
}

func pvcMetaInfoMatchDesire(pvc *corev1.PersistentVolumeClaim) bool {
	return pvc.Labels[label.ClusterIDLabelKey] == controller.TestClusterID &&
		pvc.Labels[label.MemberIDLabelKey] == controller.TestMemberID &&
		pvc.Labels[label.StoreIDLabelKey] == controller.TestStoreID &&
		pvc.Annotations[label.AnnPodNameKey] == controller.TestPodName
}

func pvMetaInfoMatchDesire(ns string, pv *corev1.PersistentVolume) bool {
	return pv.Labels[label.NamespaceLabelKey] == ns &&
		pv.Labels[label.ClusterIDLabelKey] == controller.TestClusterID &&
		pv.Labels[label.MemberIDLabelKey] == controller.TestMemberID &&
		pv.Labels[label.StoreIDLabelKey] == controller.TestStoreID &&
		pv.Annotations[label.AnnPodNameKey] == controller.TestPodName
}

func newPod(tc *v1alpha1.TidbCluster) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.TestPodName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
			Labels: map[string]string{
				label.NameLabelKey:      controller.TestName,
				label.ComponentLabelKey: controller.TestComponentName,
				label.ManagedByLabelKey: controller.TestManagedByName,
				label.InstanceLabelKey:  tc.GetLabels()[label.InstanceLabelKey],
			},
		},
		Spec: newPodSpec(v1alpha1.PDMemberType.String(), "pvc-1"),
	}
}

func newPodSpec(volumeName, pvcName string) corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "containerName",
				Image: "test",
				VolumeMounts: []corev1.VolumeMount{
					{Name: volumeName, MountPath: "/var/lib/test"},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			},
		},
	}
}
