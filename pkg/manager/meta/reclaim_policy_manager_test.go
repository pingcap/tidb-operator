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
	"fmt"
	"testing"
	"time"

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

func TestReclaimPolicyManagerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name              string
		pvcHasLabels      bool
		pvcHasVolumeName  bool
		updateErr         bool
		err               bool
		changed           bool
		enablePVRecalim   bool
		hasDeferDeleteAnn bool
	}

	testFn := func(test *testcase, t *testing.T, kind string) {
		t.Log(test.name + ": " + kind)
		var obj metav1.Object
		switch kind {
		case v1alpha1.TiDBClusterKind:
			obj = newTidbClusterForMeta()
		case v1alpha1.DMClusterKind:
			obj = newDMClusterForMeta()
		}
		pv1 := newPV("1")
		pvc1 := newPVC(obj, "1")

		if !test.pvcHasLabels {
			pvc1.Labels = nil
		}
		if !test.pvcHasVolumeName {
			pvc1.Spec.VolumeName = ""
		}
		switch kind {
		case v1alpha1.TiDBClusterKind:
			obj.(*v1alpha1.TidbCluster).Spec.EnablePVReclaim = &test.enablePVRecalim
		case v1alpha1.DMClusterKind:
			obj.(*v1alpha1.DMCluster).Spec.EnablePVReclaim = &test.enablePVRecalim
		}
		if test.hasDeferDeleteAnn {
			pvc1.Annotations = map[string]string{label.AnnPVCDeferDeleting: time.Now().String()}
		}

		rpm, fakePVControl, pvcIndexer, pvIndexer := newFakeReclaimPolicyManager()
		err := pvcIndexer.Add(pvc1)
		g.Expect(err).NotTo(HaveOccurred())
		err = pvIndexer.Add(pv1)
		g.Expect(err).NotTo(HaveOccurred())

		if test.updateErr {
			fakePVControl.SetUpdatePVError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		switch kind {
		case v1alpha1.TiDBClusterKind:
			err = rpm.Sync(obj.(*v1alpha1.TidbCluster))
		case v1alpha1.DMClusterKind:
			err = rpm.SyncDM(obj.(*v1alpha1.DMCluster))
		}
		if test.err {
			g.Expect(err).To(HaveOccurred())
			pv, err := rpm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pv.Spec.PersistentVolumeReclaimPolicy).To(Equal(corev1.PersistentVolumeReclaimDelete))
		}
		if test.changed {
			g.Expect(err).NotTo(HaveOccurred())
			pv, err := rpm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pv.Spec.PersistentVolumeReclaimPolicy).To(Equal(corev1.PersistentVolumeReclaimRetain))
		} else {
			pv, err := rpm.pvLister.Get(pv1.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pv.Spec.PersistentVolumeReclaimPolicy).To(Equal(corev1.PersistentVolumeReclaimDelete))
		}
	}

	tests := []testcase{
		{
			name:              "normal",
			pvcHasLabels:      true,
			pvcHasVolumeName:  true,
			updateErr:         false,
			err:               false,
			changed:           true,
			enablePVRecalim:   false,
			hasDeferDeleteAnn: false,
		},
		{
			name:              "pvc don't have labels",
			pvcHasLabels:      false,
			pvcHasVolumeName:  true,
			updateErr:         false,
			err:               false,
			changed:           false,
			enablePVRecalim:   false,
			hasDeferDeleteAnn: false,
		},
		{
			name:              "pvc don't have volumeName",
			pvcHasLabels:      true,
			pvcHasVolumeName:  false,
			updateErr:         false,
			err:               false,
			changed:           false,
			enablePVRecalim:   false,
			hasDeferDeleteAnn: false,
		},
		{
			name:              "enable pv reclaim and pvc has defer delete annotation",
			pvcHasLabels:      true,
			pvcHasVolumeName:  true,
			updateErr:         false,
			err:               false,
			changed:           false,
			enablePVRecalim:   true,
			hasDeferDeleteAnn: true,
		},
		{
			name:              "patch pv failed",
			pvcHasLabels:      true,
			pvcHasVolumeName:  true,
			updateErr:         true,
			err:               true,
			changed:           false,
			enablePVRecalim:   false,
			hasDeferDeleteAnn: false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t, v1alpha1.TiDBClusterKind)
		testFn(&tests[i], t, v1alpha1.DMClusterKind)
	}
}

func newFakeReclaimPolicyManager() (*reclaimPolicyManager, *controller.FakePVControl, cache.Indexer, cache.Indexer) {
	kubeCli := kubefake.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()

	pvControl := controller.NewFakePVControl(pvInformer, pvcInformer)

	return &reclaimPolicyManager{
		pvcInformer.Lister(),
		pvInformer.Lister(),
		pvControl,
	}, pvControl, pvcInformer.Informer().GetIndexer(), pvInformer.Informer().GetIndexer()
}

func newTidbClusterForMeta() *v1alpha1.TidbCluster {
	pvp := corev1.PersistentVolumeReclaimRetain
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.TestClusterName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
			Labels:    label.New().Instance(controller.TestClusterName),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PVReclaimPolicy: &pvp,
		},
	}
}

func newDMClusterForMeta() *v1alpha1.DMCluster {
	pvp := corev1.PersistentVolumeReclaimRetain
	return &v1alpha1.DMCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DMCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.TestClusterName,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
			Labels:    label.NewDM().Instance(controller.TestClusterName),
		},
		Spec: v1alpha1.DMClusterSpec{
			PVReclaimPolicy: &pvp,
		},
	}
}

func newPV(index string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pv-" + index,
			Namespace: "",
			UID:       types.UID("test"),
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			ClaimRef: &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "PersistentVolumeClaim",
				Name:       "pvc-" + index,
				Namespace:  corev1.NamespaceDefault,
				UID:        types.UID("pv" + index),
			},
		},
	}
}

func newPVC(obj metav1.Object, index string) *corev1.PersistentVolumeClaim {
	nameLabel := controller.TestName
	componentLabel := controller.TestComponentName
	if _, ok := obj.(*v1alpha1.DMCluster); ok {
		nameLabel = "dm-cluster"
		componentLabel = "dm-master"
	}

	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-" + index,
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("pvc" + index),
			Labels: map[string]string{
				label.NameLabelKey:      nameLabel,
				label.ComponentLabelKey: componentLabel,
				label.ManagedByLabelKey: controller.TestManagedByName,
				label.InstanceLabelKey:  obj.GetName(),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "pv-" + index,
		},
	}
}
