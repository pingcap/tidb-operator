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
	appslisters "k8s.io/client-go/listers/apps/v1"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestStatefulSetControlCreatesStatefulSets(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	set := newStatefulSet(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealStatefuSetControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("create", "statefulsets", func(action core.Action) (bool, runtime.Object, error) {
		create := action.(core.CreateAction)
		return true, create.GetObject(), nil
	})
	err := control.CreateStatefulSet(tc, set)
	g.Expect(err).To(Succeed())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestStatefulSetControlCreatesStatefulSetExists(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	set := newStatefulSet(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealStatefuSetControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("create", "statefulsets", func(action core.Action) (bool, runtime.Object, error) {
		return true, set, apierrors.NewAlreadyExists(action.GetResource().GroupResource(), set.Name)
	})
	err := control.CreateStatefulSet(tc, set)
	g.Expect(apierrors.IsAlreadyExists(err)).To(Equal(true))

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(0))
}

func TestStatefulSetControlCreatesStatefulSetFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	set := newStatefulSet(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealStatefuSetControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("create", "statefulsets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.CreateStatefulSet(tc, set)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}

func TestStatefulSetControlUpdateStatefulSet(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	set := newStatefulSet(tc, "pd")
	set.Spec.Replicas = func() *int32 { var i int32 = 100; return &i }()
	fakeClient := &fake.Clientset{}
	control := NewRealStatefuSetControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("update", "statefulsets", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		return true, update.GetObject(), nil
	})
	updateSS, err := control.UpdateStatefulSet(tc, set)
	g.Expect(err).To(Succeed())
	g.Expect(int(*updateSS.Spec.Replicas)).To(Equal(100))
}

func TestStatefulSetControlUpdateStatefulSetConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	set := newStatefulSet(tc, "pd")
	set.Spec.Replicas = func() *int32 { var i int32 = 100; return &i }()
	fakeClient := &fake.Clientset{}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	oldSet := newStatefulSet(tc, "pd")
	oldSet.Spec.Replicas = func() *int32 { var i int32 = 200; return &i }()
	err := indexer.Add(oldSet)
	g.Expect(err).To(Succeed())
	setLister := appslisters.NewStatefulSetLister(indexer)
	control := NewRealStatefuSetControl(fakeClient, setLister, recorder)
	conflict := false
	fakeClient.AddReactor("update", "statefulsets", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, oldSet, apierrors.NewConflict(action.GetResource().GroupResource(), set.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	updateSS, err := control.UpdateStatefulSet(tc, set)
	g.Expect(err).To(Succeed())
	g.Expect(int(*updateSS.Spec.Replicas)).To(Equal(100))
}

func TestStatefulSetControlDeleteStatefulSet(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	set := newStatefulSet(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealStatefuSetControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("delete", "statefulsets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	err := control.DeleteStatefulSet(tc, set)
	g.Expect(err).To(Succeed())
	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestStatefulSetControlDeleteStatefulSetFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	set := newStatefulSet(tc, "pd")
	fakeClient := &fake.Clientset{}
	control := NewRealStatefuSetControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("delete", "statefulsets", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(errors.New("API server down"))
	})
	err := control.DeleteStatefulSet(tc, set)
	g.Expect(err).To(HaveOccurred())

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeWarning))
}
