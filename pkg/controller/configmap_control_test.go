// Copyright 2019 PingCAP, Inc.
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
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestConfigMapControlCreatesConfigMaps(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	fakeClient := &fake.Clientset{}
	control := NewRealConfigMapControl(fakeClient, recorder)
	fakeClient.AddReactor("create", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		create := action.(core.CreateAction)
		return true, create.GetObject(), nil
	})
	_, err := control.CreateConfigMap(tc, cm)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestConfigMapControlCreatesConfigMapFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	fakeClient := &fake.Clientset{}
	control := NewRealConfigMapControl(fakeClient, recorder)
	fakeClient.AddReactor("create", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	_, err := control.CreateConfigMap(tc, cm)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestConfigMapControlUpdateConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	cm.Data["file"] = "test"
	fakeClient := &fake.Clientset{}
	control := NewRealConfigMapControl(fakeClient, recorder)
	fakeClient.AddReactor("update", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		return true, update.GetObject(), nil
	})
	updatecm, err := control.UpdateConfigMap(tc, cm)
	g.Expect(err).To(Succeed())
	g.Expect(updatecm.Data["file"]).To(Equal("test"))
}

func TestConfigMapControlUpdateConfigMapConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	cm.Data["file"] = "test"
	fakeClient := &fake.Clientset{}
	oldcm := newConfigMap()
	oldcm.Data["file"] = "test2"
	control := NewRealConfigMapControl(fakeClient, recorder)
	conflict := false
	fakeClient.AddReactor("update", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, oldcm, apierrors.NewConflict(action.GetResource().GroupResource(), cm.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	updatecm, err := control.UpdateConfigMap(tc, cm)
	g.Expect(err).To(Succeed())
	g.Expect(updatecm.Data["file"]).To(Equal("test"))
}

func TestConfigMapControlDeleteConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	fakeClient := &fake.Clientset{}
	control := NewRealConfigMapControl(fakeClient, recorder)
	fakeClient.AddReactor("delete", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.DeleteConfigMap(tc, cm)
	g.Expect(err).To(Succeed())
	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestConfigMapControlDeleteConfigMapFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	fakeClient := &fake.Clientset{}
	control := NewRealConfigMapControl(fakeClient, recorder)
	fakeClient.AddReactor("delete", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.DeleteConfigMap(tc, cm)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func newConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Data: map[string]string{},
	}
}
