// Copyright 2019. PingCAP, Inc.
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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestConfigMapControlApplyConfigMaps(t *testing.T) {
	g := NewGomegaWithT(t)

	type testCase struct {
		name             string
		getOld           func() *corev1.ConfigMap
		getNew           func() *corev1.ConfigMap
		errOnCreate      bool
		errOnUpdate      bool
		conflictOnUpdate bool
		expectFn         func(*GomegaWithT, *corev1.ConfigMap, *fake.Clientset, error)
	}
	testFn := func(tt *testCase) {
		t.Log(tt.name)

		recorder := record.NewFakeRecorder(10)
		tc := newTidbCluster()
		fakeClient := &fake.Clientset{}
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		oldCm := tt.getOld()
		if oldCm != nil {
			err := indexer.Add(oldCm)
			g.Expect(err).To(Succeed())
			_, err = fakeClient.CoreV1().ConfigMaps("default").Create(oldCm)
			g.Expect(err).To(Succeed())
			fakeClient.ClearActions()
		}

		cmLister := corelisters.NewConfigMapLister(indexer)
		control := NewRealConfigMapControl(fakeClient, cmLister, recorder)
		newCm := tt.getNew()
		if tt.errOnCreate {
			fakeClient.AddReactor("create", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("API server down")
			})
		}
		if tt.errOnUpdate {
			fakeClient.AddReactor("update", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("API server down")
			})
		}
		if tt.conflictOnUpdate {
			fakeClient.AddReactor("update", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewConflict(action.GetResource().GroupResource(), newCm.Name, errors.New("conflict"))
			})
		}
		fakeClient.AddReactor("create", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
			if oldCm != nil {
				return true, nil, apierrors.NewAlreadyExists(action.GetResource().GroupResource(), oldCm.Name)
			}
			create := action.(core.CreateAction)
			return true, create.GetObject(), nil
		})
		applied, err := control.ApplyConfigMap(tc, newCm)

		tt.expectFn(g, applied, fakeClient, err)
	}
	cases := []*testCase{
		{
			name:        "apply on absent",
			getOld:      func() *corev1.ConfigMap { return nil },
			getNew:      func() *corev1.ConfigMap { return newConfigMap() },
			errOnCreate: false,
			errOnUpdate: false,
			expectFn: func(g *GomegaWithT, cm *corev1.ConfigMap, fakeCli *fake.Clientset, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(fakeCli.Actions()[0].GetVerb()).To(Equal("create"))
			},
		},
		{
			name: "apply on existing",
			getOld: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test"
				return cm
			},
			getNew: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test2"
				return cm
			},
			errOnCreate: false,
			errOnUpdate: false,
			expectFn: func(g *GomegaWithT, cm *corev1.ConfigMap, fakeCli *fake.Clientset, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(fakeCli.Actions()[0].GetVerb()).To(Equal("create"))
				g.Expect(fakeCli.Actions()[1].GetVerb()).To(Equal("update"))
			},
		},
		{
			name: "apply on existing no change",
			getOld: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test"
				return cm
			},
			getNew: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test"
				return cm
			},
			errOnCreate: false,
			errOnUpdate: false,
			expectFn: func(g *GomegaWithT, cm *corev1.ConfigMap, fakeCli *fake.Clientset, err error) {
				g.Expect(err).To(Succeed())
				g.Expect(fakeCli.Actions()[0].GetVerb()).To(Equal("create"))
				g.Expect(fakeCli.Actions()).To(HaveLen(1))
			},
		},
		{
			name: "apply on existing conflict",
			getOld: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test"
				return cm
			},
			getNew: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test1"
				return cm
			},
			errOnCreate:      false,
			errOnUpdate:      false,
			conflictOnUpdate: true,
			expectFn: func(g *GomegaWithT, cm *corev1.ConfigMap, fakeCli *fake.Clientset, err error) {
				g.Expect(err).To(WithTransform(IsRequeueError, BeTrue()))
				g.Expect(fakeCli.Actions()[0].GetVerb()).To(Equal("create"))
				g.Expect(fakeCli.Actions()[1].GetVerb()).To(Equal("update"))
			},
		},
		{
			name: "update error during apply",
			getOld: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test"
				return cm
			},
			getNew: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test1"
				return cm
			},
			errOnCreate: false,
			errOnUpdate: true,
			expectFn: func(g *GomegaWithT, cm *corev1.ConfigMap, fakeCli *fake.Clientset, err error) {
				g.Expect(err).To(WithTransform(IsRequeueError, BeTrue()))
				g.Expect(fakeCli.Actions()[0].GetVerb()).To(Equal("create"))
				g.Expect(fakeCli.Actions()[1].GetVerb()).To(Equal("update"))
			},
		},
		{
			name:   "create error during apply",
			getOld: func() *corev1.ConfigMap { return nil },
			getNew: func() *corev1.ConfigMap {
				cm := newConfigMap()
				cm.Data["file"] = "test1"
				return cm
			},
			errOnCreate: true,
			errOnUpdate: false,
			expectFn: func(g *GomegaWithT, cm *corev1.ConfigMap, fakeCli *fake.Clientset, err error) {
				g.Expect(err).To(WithTransform(IsRequeueError, BeTrue()))
				g.Expect(fakeCli.Actions()[0].GetVerb()).To(Equal("create"))
			},
		},
	}

	for _, tt := range cases {
		testFn(tt)
	}
}

func TestConfigMapControlCreatesConfigMaps(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	fakeClient := &fake.Clientset{}
	control := NewRealConfigMapControl(fakeClient, nil, recorder)
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
	control := NewRealConfigMapControl(fakeClient, nil, recorder)
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
	control := NewRealConfigMapControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("update", "configmaps", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		return true, update.GetObject(), nil
	})
	updatecm, err := control.UpdateConfigMap(tc, cm)
	g.Expect(err).To(Succeed())
	g.Expect(updatecm.Data["file"]).To(Equal("test"))

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestConfigMapControlUpdateConfigMapConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	cm.Data["file"] = "test"
	fakeClient := &fake.Clientset{}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	oldcm := newConfigMap()
	oldcm.Data["file"] = "test2"
	err := indexer.Add(oldcm)
	g.Expect(err).To(Succeed())
	cmLister := corelisters.NewConfigMapLister(indexer)
	control := NewRealConfigMapControl(fakeClient, cmLister, recorder)
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

	events := collectEvents(recorder.Events)
	g.Expect(events).To(HaveLen(1))
	g.Expect(events[0]).To(ContainSubstring(corev1.EventTypeNormal))
}

func TestConfigMapControlDeleteConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTidbCluster()
	cm := newConfigMap()
	fakeClient := &fake.Clientset{}
	control := NewRealConfigMapControl(fakeClient, nil, recorder)
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
	control := NewRealConfigMapControl(fakeClient, nil, recorder)
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
