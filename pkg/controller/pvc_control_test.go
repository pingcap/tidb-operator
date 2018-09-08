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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestPVCControlUpdateMetaInfoSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pvc := newPVC(tc)
	pod := newPod(tc)
	fakeClient, pvcLister, recorder := newFakeClientAndRecorder()
	control := NewRealPVCControl(fakeClient, recorder, pvcLister)
	fakeClient.AddReactor("update", "persistentvolumeclaims", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.UpdateMetaInfo(tc, pvc, pod)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestPVCControlUpdateMetaInfoFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pvc := newPVC(tc)
	pod := newPod(tc)
	fakeClient, pvcLister, recorder := newFakeClientAndRecorder()
	control := NewRealPVCControl(fakeClient, recorder, pvcLister)
	fakeClient.AddReactor("update", "persistentvolumeclaims", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.UpdateMetaInfo(tc, pvc, pod)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestPVCControlUpdateMetaInfoConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	pvc := newPVC(tc)
	pod := newPod(tc)
	fakeClient, pvcLister, recorder := newFakeClientAndRecorder()
	control := NewRealPVCControl(fakeClient, recorder, pvcLister)
	conflict := false
	fakeClient.AddReactor("update", "persistentvolumeclaims", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, update.GetObject(), apierrors.NewConflict(action.GetResource().GroupResource(), pvc.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	err := control.UpdateMetaInfo(tc, pvc, pod)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func newFakeClientAndRecorder() (*fake.Clientset, corelisters.PersistentVolumeClaimLister, *record.FakeRecorder) {
	kubeCli := &fake.Clientset{}
	recorder := record.NewFakeRecorder(10)
	pvcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().PersistentVolumeClaims()
	return kubeCli, pvcInformer.Lister(), recorder
}

func newPVC(tc *v1alpha1.TidbCluster) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-1",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
			Labels: map[string]string{
				label.ComponentLabelKey: TestComponentName,
				label.ManagedByLabelKey: TestManagedByName,
				label.InstanceLabelKey:  tc.GetName(),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "pv-1",
		},
	}
}
