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
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestBackupControlCreateBackupSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	backup := newBackup()
	fakeClient := &fake.Clientset{}
	control := NewRealBackupControl(fakeClient, recorder)
	fakeClient.AddReactor("create", "backups", func(action core.Action) (bool, runtime.Object, error) {
		create := action.(core.CreateAction)
		return true, create.GetObject(), nil
	})
	_, err := control.CreateBackup(backup)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestBackupControlCreateBackupFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	backup := newBackup()
	fakeClient := &fake.Clientset{}
	control := NewRealBackupControl(fakeClient, recorder)
	fakeClient.AddReactor("create", "backups", func(action core.Action) (bool, runtime.Object, error) {
		create := action.(core.CreateAction)
		return true, create.GetObject(), apierrors.NewInternalError(errors.New("API server down"))
	})
	_, err := control.CreateBackup(backup)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestBackupControlDeleteBackupSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	backup := newBackup()
	fakeClient := &fake.Clientset{}
	control := NewRealBackupControl(fakeClient, recorder)
	fakeClient.AddReactor("delete", "backups", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.DeleteBackup(backup)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestBackupControlDeleteBackupFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	backup := newBackup()
	fakeClient := &fake.Clientset{}
	control := NewRealBackupControl(fakeClient, recorder)
	fakeClient.AddReactor("delete", "backups", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.DeleteBackup(backup)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}
