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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

func TestJobControlCreatesJobsSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	backup := newBackup()
	job := newJobFromBackup(backup)
	fakeClient := &fake.Clientset{}
	control := NewRealJobControl(fakeClient, recorder)
	fakeClient.AddReactor("create", "jobs", func(action core.Action) (bool, runtime.Object, error) {
		create := action.(core.CreateAction)
		return true, create.GetObject(), nil
	})
	err := control.CreateJob(backup, job)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestJobControlCreatesJobFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	backup := newBackup()
	job := newJobFromBackup(backup)
	fakeClient := &fake.Clientset{}
	control := NewRealJobControl(fakeClient, recorder)
	fakeClient.AddReactor("create", "jobs", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.CreateJob(backup, job)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestJobControlDeleteJobSuccess(t *testing.T) {
	for _, tc := range []struct {
		name string
		uid  types.UID
	}{
		{name: "with UID", uid: "job-uid"},
		{name: "without UID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			recorder := record.NewFakeRecorder(10)
			backup := newBackup()
			job := newJobFromBackup(backup)
			job.UID = tc.uid
			fakeClient := &fake.Clientset{}
			control := NewRealJobControl(fakeClient, recorder)
			fakeClient.AddReactor("delete", "jobs", func(_ core.Action) (bool, runtime.Object, error) {
				return true, nil, nil
			})
			err := control.DeleteJob(backup, job)
			g.Expect(err).To(Succeed())

			actions := fakeClient.Actions()
			g.Expect(actions).To(HaveLen(1))
			opts := actions[0].(core.DeleteAction).GetDeleteOptions()
			propagation := metav1.DeletePropagationForeground
			g.Expect(opts.PropagationPolicy).To(Equal(&propagation))
			if tc.uid == "" {
				g.Expect(opts.Preconditions).To(BeNil())
			} else {
				g.Expect(opts.Preconditions).To(Equal(&metav1.Preconditions{UID: &tc.uid}))
			}

			events := collectEvents(recorder.Events)
			g.Expect(events).To(HaveLen(1))
			g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
		})
	}
}

func TestJobControlDeleteJobFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	backup := newBackup()
	job := newJobFromBackup(backup)
	fakeClient := &fake.Clientset{}
	control := NewRealJobControl(fakeClient, recorder)
	fakeClient.AddReactor("delete", "jobs", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.DeleteJob(backup, job)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}
