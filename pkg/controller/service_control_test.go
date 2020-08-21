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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestServiceControlCreatesServices(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	svc := newService(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealServiceControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("create", "services", func(action core.Action) (bool, runtime.Object, error) {
		create := action.(core.CreateAction)
		return true, create.GetObject(), nil
	})
	err := control.CreateService(tc, svc)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestServiceControlCreatesServiceFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	svc := newService(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealServiceControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("create", "services", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.CreateService(tc, svc)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestServiceControlUpdateService(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	svc := newService(tc, "pd")
	svc.Spec.ClusterIP = "1.1.1.1"
	fakeClient := &fake.Clientset{}
	control := NewRealServiceControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("update", "services", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		return true, update.GetObject(), nil
	})
	updateSvc, err := control.UpdateService(tc, svc)
	g.Expect(err).To(Succeed())
	g.Expect(updateSvc.Spec.ClusterIP).To(Equal("1.1.1.1"))
}

func TestServiceControlUpdateServiceConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	svc := newService(tc, "pd")
	svc.Spec.ClusterIP = "1.1.1.1"
	fakeClient := &fake.Clientset{}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	oldSvc := newService(tc, "pd")
	oldSvc.Spec.ClusterIP = "2.2.2.2"
	err := indexer.Add(oldSvc)
	g.Expect(err).To(Succeed())
	svcLister := corelisters.NewServiceLister(indexer)
	control := NewRealServiceControl(fakeClient, svcLister, recorder)
	conflict := false
	fakeClient.AddReactor("update", "services", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, oldSvc, apierrors.NewConflict(action.GetResource().GroupResource(), svc.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	updateSvc, err := control.UpdateService(tc, svc)
	g.Expect(err).To(Succeed())
	g.Expect(updateSvc.Spec.ClusterIP).To(Equal("1.1.1.1"))
}

func TestServiceControlDeleteService(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	svc := newService(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealServiceControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("delete", "services", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.DeleteService(tc, svc)
	g.Expect(err).To(Succeed())
	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestServiceControlDeleteServiceFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	svc := newService(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealServiceControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("delete", "services", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.DeleteService(tc, svc)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}
