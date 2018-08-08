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

package controller

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestPVControlPatchPVReclaimPolicySuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	pvcInformer, recorder := newFakeRecorderAndPVCInformer()
	tc := newTidbCluster()
	pv := newPV()
	fakeClient := &fake.Clientset{}
	control := NewRealPVControl(fakeClient, pvcInformer.Lister(), recorder)
	fakeClient.AddReactor("patch", "persistentvolumes", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.PatchPVReclaimPolicy(tc, pv, tc.Spec.PVReclaimPolicy)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestPVControlPatchPVReclaimPolicyFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	pvcInformer, recorder := newFakeRecorderAndPVCInformer()
	tc := newTidbCluster()
	pv := newPV()
	fakeClient := &fake.Clientset{}
	control := NewRealPVControl(fakeClient, pvcInformer.Lister(), recorder)
	fakeClient.AddReactor("patch", "persistentvolumes", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.PatchPVReclaimPolicy(tc, pv, tc.Spec.PVReclaimPolicy)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestPVControlPatchPVReclaimPolicyConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	pvcInformer, recorder := newFakeRecorderAndPVCInformer()
	tc := newTidbCluster()
	pv := newPV()
	fakeClient := &fake.Clientset{}
	control := NewRealPVControl(fakeClient, pvcInformer.Lister(), recorder)

	conflict := false
	fakeClient.AddReactor("patch", "persistentvolumes", func(action core.Action) (bool, runtime.Object, error) {
		if !conflict {
			conflict = true
			return true, nil, apierrors.NewConflict(action.GetResource().GroupResource(), pv.Name, errors.New("conflict"))
		}
		return true, nil, nil
	})
	err := control.PatchPVReclaimPolicy(tc, pv, tc.Spec.PVReclaimPolicy)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func newFakeRecorderAndPVCInformer() (coreinformers.PersistentVolumeClaimInformer, *record.FakeRecorder) {
	kubeCli := kubefake.NewSimpleClientset()
	pvcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().PersistentVolumeClaims()
	recorder := record.NewFakeRecorder(10)
	return pvcInformer, recorder
}

func newPV() *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pv-1",
			Namespace: "",
			UID:       types.UID("test"),
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
		},
	}
}
