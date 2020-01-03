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

package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/apis/example"
	examplev1 "k8s.io/apiserver/pkg/apis/example/v1"
	"k8s.io/apiserver/pkg/storage"
	core "k8s.io/client-go/testing"
)

const (
	storageResource = "dataresources"
)

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	metav1.AddToGroupVersion(scheme, metav1.SchemeGroupVersion)
	utilruntime.Must(example.AddToScheme(scheme))
	utilruntime.Must(examplev1.AddToScheme(scheme))
}

func TestCreate(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name        string
		obj         *example.Pod
		errOnAction func(action core.Action) (bool, runtime.Object, error)
		preData     []*example.Pod
		expectFn    func(*GomegaWithT, []core.Action, []v1alpha1.DataResource, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, fakeCli, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}
		// clear prepare actions
		fakeCli.ClearActions()
		if test.errOnAction != nil {
			fakeCli.PrependReactor("create", storageResource, test.errOnAction)
		}
		err := store.Create(ctx, keyFunc(test.obj), test.obj, out, 0)
		// record actions
		actions := fakeCli.Actions()

		l, _ := fakeCli.PingcapV1alpha1().DataResources("default").List(metav1.ListOptions{})
		test.expectFn(g, actions, l.Items, err)
	}

	tests := []testcase{
		{
			name:        "normal creation",
			obj:         &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			errOnAction: nil,
			preData:     nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(actions).Should(HaveLen(1))
				expectActions(g, actions, 1, "create", storageResource)
				g.Expect(remains).Should(HaveLen(1))
			},
		},
		{
			name:        "duplicated creation",
			obj:         &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			errOnAction: nil,
			preData:     []*example.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}},
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).Should(HaveOccurred(), "Expected error when creating duplicated objects")
				g.Expect(remains).Should(HaveLen(1))
			},
		},
		{
			name:        "creation with no duplicate",
			obj:         &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			errOnAction: nil,
			preData:     []*example.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "bar"}}},
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(remains).Should(HaveLen(2))
			},
		},
		{
			name: "error on creation",
			obj:  &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			errOnAction: func(action core.Action) (b bool, object runtime.Object, e error) {
				return true, nil, fmt.Errorf("Error occurs when creating Pod")
			},
			preData: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(remains).Should(BeEmpty(), "Expected no objects created when error occurs on creation")
			},
		},
		{
			name: "UID consistency",
			obj: &example.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					UID:  "1",
				},
			},
			errOnAction: nil,
			preData:     nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(remains).Should(HaveLen(1))
				g.Expect(string(remains[0].UID)).Should(Equal("1"), "Expected the UID of storage resource is equal to the object")
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestDelete(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name        string
		obj         *example.Pod
		errOnAction func(action core.Action) (bool, runtime.Object, error)
		preData     []*example.Pod
		expectFn    func(*GomegaWithT, []core.Action, []v1alpha1.DataResource, error)

		precondition *storage.Preconditions
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, fakeCli, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}
		// clear prepare actions
		fakeCli.ClearActions()
		if test.errOnAction != nil {
			fakeCli.PrependReactor("delete", storageResource, test.errOnAction)
		}
		err := store.Delete(ctx, keyFunc(test.obj), out, test.precondition, nil)
		// record actions
		actions := fakeCli.Actions()

		l, _ := fakeCli.PingcapV1alpha1().DataResources("default").List(metav1.ListOptions{})
		test.expectFn(g, actions, l.Items, err)
	}

	preData := []*example.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
				UID:  "1",
			},
		},
	}
	recentUID := types.UID("1")
	// staleUID := types.UID("0")
	tests := []testcase{
		{
			name:         "delete without precondition",
			preData:      preData,
			obj:          preData[0],
			errOnAction:  nil,
			precondition: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(actions).Should(HaveLen(1))
				expectActions(g, actions, 1, "delete", storageResource)
				g.Expect(remains).Should(HaveLen(len(preData) - 1))
			},
		},
		{
			name:        "delete with up-to-date UID",
			preData:     preData,
			obj:         preData[0],
			errOnAction: nil,
			precondition: &storage.Preconditions{
				UID: &recentUID,
			},
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(remains).Should(HaveLen(len(preData) - 1))
			},
		},
		// Fake client do not respect UID
		// TODO: mock UID precondition check on our own
		//{
		//	name:        "delete with stale UID",
		//	preData:     preData,
		//	obj:         preData[0],
		//	errOnAction: nil,
		//	precondition: &storage.Preconditions{
		//		UID: &staleUID,
		//	},
		//	expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
		//		g.Expect(err).ShouldNot(HaveOccurred())
		//		g.Expect(remains).Should(HaveLen(len(preData)), "Expected the object not getting deleted when delete with a stale UID")
		//	},
		//},
		{
			name:        "delete an object that does not exist",
			preData:     nil,
			obj:         preData[0],
			errOnAction: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).Should(HaveOccurred(), "Expected error when delete objects that does not exist")
			},
		},
		{
			name:    "error on deletion",
			preData: preData,
			obj:     preData[0],
			errOnAction: func(action core.Action) (b bool, object runtime.Object, e error) {
				return true, nil, errors.Errorf("Error when deleting object")
			},
			expectFn: func(g *GomegaWithT, actions []core.Action, remains []v1alpha1.DataResource, err error) {
				g.Expect(err).Should(HaveOccurred(), "Expected error when delete objects that does not exist")
				g.Expect(remains).Should(HaveLen(len(preData)), "Expected the object not getting  deleted when error occurs upon deletion")
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestGet(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		obj      *example.Pod
		preData  []*example.Pod
		expectFn func(*GomegaWithT, []core.Action, *example.Pod, error)

		ignoreNotFound bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, fakeCli, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}
		// clear prepare actions
		fakeCli.ClearActions()
		result := &example.Pod{}
		err := store.Get(ctx, keyFunc(test.obj), "", result, test.ignoreNotFound)
		// record actions
		actions := fakeCli.Actions()

		test.expectFn(g, actions, result, err)
	}
	preData := []*example.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
				UID:  "1",
			},
			Spec: example.PodSpec{
				NodeName: "test",
			},
		},
	}
	tests := []testcase{
		{
			name:           "get objects that exist",
			preData:        preData,
			obj:            preData[0],
			ignoreNotFound: false,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.Pod, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Spec.NodeName).Should(Equal(preData[0].Spec.NodeName))
			},
		},
		{
			name:           "get objects that does not exist",
			preData:        preData,
			obj:            &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
			ignoreNotFound: false,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.Pod, err error) {
				g.Expect(err).Should(HaveOccurred(), "Expected error when get object that does not exist with ignoreNotFound=false")
				g.Expect(result.Name).Should(BeEmpty())
			},
		},
		{
			name:           "get objects that does not exist with ignoreNotFound",
			preData:        preData,
			obj:            &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
			ignoreNotFound: true,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.Pod, err error) {
				g.Expect(err).ShouldNot(HaveOccurred(), "Expected no error when get object that does not exist with ignoreNotFound=true")
				g.Expect(result.Name).Should(BeEmpty())
			},
		},
		// TODO: new case - error on get
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestGetToList(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name     string
		obj      *example.Pod
		preData  []*example.Pod
		expectFn func(*GomegaWithT, []core.Action, *example.PodList, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, fakeCli, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}
		// clear prepare actions
		fakeCli.ClearActions()
		result := &example.PodList{}
		err := store.GetToList(ctx, keyFunc(test.obj), "", predictionFunc(labels.Everything(), fields.Everything()), result)
		// record actions
		actions := fakeCli.Actions()

		test.expectFn(g, actions, result, err)
	}
	preData := []*example.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
				UID:  "1",
			},
			Spec: example.PodSpec{
				NodeName: "test",
			},
		},
	}
	tests := []testcase{
		{
			name:    "get objects that exist to list",
			preData: preData,
			obj:     preData[0],
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).ShouldNot(BeEmpty())
				g.Expect(result.Items[0].Spec.NodeName).Should(Equal(preData[0].Spec.NodeName))
			},
		},
		{
			name:    "get objects that does not exist to list",
			preData: preData,
			obj:     &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bar"}},
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).Should(BeEmpty())
			},
		},
		// TODO: new case - error on getToList
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestList(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name          string
		namespace     string
		labelSelector labels.Selector
		fieldSelector fields.Selector
		preData       []*example.Pod
		expectFn      func(*GomegaWithT, []core.Action, *example.PodList, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, fakeCli, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}

		// clear prepare actions
		fakeCli.ClearActions()
		result := &example.PodList{}
		labelSelector := labels.Everything()
		if test.labelSelector != nil {
			labelSelector = test.labelSelector
		}
		fieldSelector := fields.Everything()
		if test.fieldSelector != nil {
			fieldSelector = test.fieldSelector
		}
		err := store.List(ctx, listKeyFunc(test.namespace), "", predictionFunc(labelSelector, fieldSelector), result)
		// record actions
		actions := fakeCli.Actions()

		test.expectFn(g, actions, result, err)
	}
	preData := []*example.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "ns1",
				UID:       "1",
				Labels: map[string]string{
					"name": "foo",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bar",
				Namespace: "ns1",
				UID:       "1",
				Labels: map[string]string{
					"name": "bar",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bar",
				Namespace: "ns2",
				UID:       "1",
				Labels: map[string]string{
					"name": "bar",
				},
			},
		},
	}
	tests := []testcase{
		{
			name:          "list objects in all namespaces",
			preData:       preData,
			namespace:     "",
			labelSelector: nil,
			fieldSelector: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).Should(HaveLen(len(preData)))
			},
		},
		{
			name:          "list objects in specific namespace",
			preData:       preData,
			namespace:     "ns1",
			labelSelector: nil,
			fieldSelector: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).ShouldNot(BeEmpty())
				for _, item := range result.Items {
					g.Expect(item.Namespace).Should(Equal("ns1"))
				}
			},
		},
		{
			name:          "list an empty namespace",
			preData:       preData,
			namespace:     "empty",
			labelSelector: nil,
			fieldSelector: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).Should(BeEmpty())
			},
		},
		{
			name:      "list with label selector that match objects",
			preData:   preData,
			namespace: "",
			labelSelector: labels.SelectorFromSet(labels.Set(map[string]string{
				"name": "bar",
			})),
			fieldSelector: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).ShouldNot(BeEmpty())
				for _, item := range result.Items {
					g.Expect(item.Labels["name"]).Should(Equal("bar"))
				}
			},
		},
		{
			name:      "list with label selector that does not match any objects",
			preData:   preData,
			namespace: "",
			labelSelector: labels.SelectorFromSet(labels.Set(map[string]string{
				"name": "not-exists",
			})),
			fieldSelector: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).Should(BeEmpty())
			},
		},
		{
			name:          "list with field selector that match objects",
			preData:       preData,
			namespace:     "",
			labelSelector: nil,
			fieldSelector: fields.SelectorFromSet(fields.Set(map[string]string{
				"metadata.name": "bar",
			})),
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).ShouldNot(BeEmpty())
				for _, item := range result.Items {
					g.Expect(item.Name).Should(Equal("bar"))
				}
			},
		},
		{
			name:          "list with field selector that does not match any objects",
			preData:       preData,
			namespace:     "",
			labelSelector: nil,
			fieldSelector: fields.SelectorFromSet(fields.Set(map[string]string{
				"metadata.name": "not-exists",
			})),
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.PodList, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(result.Items).Should(BeEmpty())
			},
		},
		// TODO: new case - error on list
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestListShouldReturnFullList(t *testing.T) {
	g := NewGomegaWithT(t)
	store, _, ctx := testSetup(t)

	pred := predictionFunc(labels.Everything(), fields.Everything())
	pred.Limit = 5
	testFunc := func() {
		result := &example.PodList{}
		err := store.List(ctx, listKeyFunc("default"), "", pred, result)
		g.Expect(err).ShouldNot(HaveOccurred(), "list resources")
		g.Expect(result.ListMeta.Continue).Should(BeEmpty(), "Expected full list returned")
		g.Expect(result.ListMeta.RemainingItemCount).Should(BeNil(), "Expected full list returned and no items remaining")
	}

	// empty list
	testFunc()

	// list size do not exceed limit
	out := &example.Pod{}
	in := &example.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	err := store.Create(ctx, keyFunc(in), in, out, 0)
	g.Expect(err).ShouldNot(HaveOccurred(), "create a new resource")
	testFunc()

	// list size exceed limit
	for i := 0; i < 10; i++ {
		in := &example.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("foo-%d", i),
			},
		}
		err := store.Create(ctx, keyFunc(in), in, out, 0)
		g.Expect(err).ShouldNot(HaveOccurred(), "create a new resource")
	}
	testFunc()
}

// TODO: conflict case and suggestion case
func TestGuaranteedUpdate(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name           string
		preData        []*example.Pod
		upgraded       *example.Pod
		ignoreNotFound bool
		errOnAction    func(action core.Action) (bool, runtime.Object, error)
		// TODO: precondition case
		precondition *storage.Preconditions
		expectFn     func(*GomegaWithT, []core.Action, *example.Pod, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, fakeCli, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}
		// clear prepare actions
		fakeCli.ClearActions()
		if test.errOnAction != nil {
			fakeCli.PrependReactor("update", storageResource, test.errOnAction)
		}
		tryUpdate := func(input runtime.Object, res storage.ResponseMeta) (output runtime.Object, ttl *uint64, err error) {
			defaultTTL := uint64(0)
			return test.upgraded, &defaultTTL, nil
		}
		err := store.GuaranteedUpdate(ctx, keyFunc(test.upgraded), out, test.ignoreNotFound, test.precondition, tryUpdate)
		// record actions
		actions := fakeCli.Actions()

		result := &example.Pod{}
		store.Get(ctx, keyFunc(test.upgraded), "", result, false)

		test.expectFn(g, actions, result, err)
	}
	preData := []*example.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: example.PodSpec{
				ServiceAccountName: "test",
			},
		},
	}
	upgraded := preData[0].DeepCopy()
	upgraded.Spec.ServiceAccountName = "upgraded"
	tests := []testcase{
		{
			name:           "update existing resource",
			preData:        preData,
			upgraded:       upgraded,
			ignoreNotFound: false,
			errOnAction:    nil,
			precondition:   nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.Pod, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(actions).Should(HaveLen(2))
				expectActions(g, actions, 1, "update", storageResource)
				g.Expect(result.Spec.ServiceAccountName).Should(Equal(upgraded.Spec.ServiceAccountName))
			},
		},
		{
			name:           "update resource that does not exist",
			preData:        preData,
			upgraded:       &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "not-exist"}},
			ignoreNotFound: false,
			errOnAction:    nil,
			precondition:   nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.Pod, err error) {
				g.Expect(err).Should(HaveOccurred(), "Expected an error when update a resource that does not exist with ignoreNotFound=false")
			},
		},
		{
			name:           "update resource that does not exist with ignoreNotFound",
			preData:        preData,
			upgraded:       &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "not-exist"}},
			ignoreNotFound: true,
			errOnAction:    nil,
			precondition:   nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.Pod, err error) {
				g.Expect(err).ShouldNot(HaveOccurred(), "Expected no error when update a resource that does not exist with ignoreNotFound=true")
			},
		},
		{
			name:           "error on update",
			preData:        preData,
			upgraded:       upgraded,
			ignoreNotFound: false,
			errOnAction: func(action core.Action) (b bool, object runtime.Object, e error) {
				return true, nil, fmt.Errorf("Error occurred on updating")
			},
			precondition: nil,
			expectFn: func(g *GomegaWithT, actions []core.Action, result *example.Pod, err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(result.Spec.ServiceAccountName).Should(Equal(preData[0].Spec.ServiceAccountName))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestWatch(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name      string
		preData   []*example.Pod
		obj       *example.Pod
		mutations func(store storage.Interface, ctx context.Context)
		expectFn  func(*GomegaWithT, []watch.Event, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, _, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}
		watcher, err := store.Watch(ctx, keyFunc(test.obj), "", predictionFunc(labels.Everything(), fields.Everything()))
		var events []watch.Event
		var mutationFinish sync.WaitGroup
		mutationFinish.Add(1)
		if err == nil {
			go func() {
				test.mutations(store, ctx)
				time.Sleep(time.Second)
				watcher.Stop()
				mutationFinish.Done()
			}()
			for event := range watcher.ResultChan() {
				events = append(events, event)
			}
			mutationFinish.Wait()
		}

		test.expectFn(g, events, err)
	}
	preData := []*example.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
			Spec: example.PodSpec{
				ServiceAccountName: "test",
			},
		},
	}
	tests := []testcase{
		{
			name:    "watch CRUD",
			preData: preData,
			obj:     preData[0],
			mutations: func(store storage.Interface, ctx context.Context) {
				// upgrade
				out := &example.Pod{}
				to := preData[0].DeepCopy()
				to.Spec.ServiceAccountName = "upgraded"
				tryUpdate := func(input runtime.Object, res storage.ResponseMeta) (output runtime.Object, ttl *uint64, err error) {
					defaultTTL := uint64(0)
					return to, &defaultTTL, nil
				}
				if err := store.GuaranteedUpdate(ctx, keyFunc(preData[0]), out, false, nil, tryUpdate); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}

				// delete
				if err := store.Delete(ctx, keyFunc(preData[0]), out, nil, nil); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}

				// re-create
				if err := store.Create(ctx, keyFunc(preData[0]), to, out, 0); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}
			},
			expectFn: func(g *GomegaWithT, events []watch.Event, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(events).Should(HaveLen(3))
				g.Expect(events[0].Type).Should(Equal(watch.Modified))
				g.Expect(events[1].Type).Should(Equal(watch.Deleted))
				g.Expect(events[2].Type).Should(Equal(watch.Added))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestWatchList(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name      string
		namespace string
		pred      storage.SelectionPredicate
		preData   []*example.Pod
		mutations func(store storage.Interface, ctx context.Context)
		expectFn  func(*GomegaWithT, []watch.Event, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		store, _, ctx := testSetup(t)

		out := &example.Pod{}
		for _, po := range test.preData {
			if err := store.Create(ctx, keyFunc(po), po, out, 0); err != nil {
				t.Errorf("Expected no error when preparing predata, got %v", err)
			}
		}
		watcher, err := store.WatchList(ctx, listKeyFunc(test.namespace), "", test.pred)
		var events []watch.Event
		var mutationFinish sync.WaitGroup
		mutationFinish.Add(1)
		if err == nil {
			go func() {
				test.mutations(store, ctx)
				time.Sleep(time.Second)
				watcher.Stop()
				mutationFinish.Done()
			}()
			for event := range watcher.ResultChan() {
				events = append(events, event)
			}
			mutationFinish.Wait()
		}

		test.expectFn(g, events, err)
	}
	preData := []*example.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns1",
				Name:      "foo",
				Labels: map[string]string{
					"label1": "v1",
				},
			},
			Spec: example.PodSpec{
				ServiceAccountName: "test",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns1",
				Name:      "bar",
				Labels: map[string]string{
					"label1": "v2",
				},
			},
			Spec: example.PodSpec{
				ServiceAccountName: "test",
			},
		},
	}
	tests := []testcase{
		{
			name:      "watch list",
			preData:   preData,
			namespace: "ns1",
			pred:      predictionFunc(labels.Everything(), fields.Everything()),
			mutations: func(store storage.Interface, ctx context.Context) {
				out := &example.Pod{}
				// delete
				if err := store.Delete(ctx, keyFunc(preData[0]), out, nil, nil); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}

				toCreate := &example.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "newly-created",
					},
				}
				// create a new one
				if err := store.Create(ctx, keyFunc(toCreate), toCreate, out, 0); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}
			},
			expectFn: func(g *GomegaWithT, events []watch.Event, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(events).Should(HaveLen(2))
				g.Expect(events[0].Type).Should(Equal(watch.Deleted))
				g.Expect(events[1].Type).Should(Equal(watch.Added))
			},
		},
		{
			name:      "watch list with prediction",
			namespace: "ns1",
			preData:   preData,
			pred: predictionFunc(
				labels.SelectorFromSet(labels.Set(map[string]string{
					"label1": "v1",
				})), fields.Everything(),
			),
			mutations: func(store storage.Interface, ctx context.Context) {
				out := &example.Pod{}
				matchLabel := &example.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "newly-created2",
						Labels: map[string]string{
							"label1": "v1",
						},
					},
				}
				// create a new pod that does not match the label selector
				if err := store.Create(ctx, keyFunc(matchLabel), matchLabel, out, 0); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}

				notMatchLabel := &example.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "newly-created1",
						Labels: map[string]string{
							"label1": "v2",
						},
					},
				}
				// create a new pod that does not match the label selector
				if err := store.Create(ctx, keyFunc(notMatchLabel), notMatchLabel, out, 0); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}

				// delete pod that does not match label selector
				if err := store.Delete(ctx, keyFunc(preData[1]), out, nil, nil); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}

				// delete pod that match label selector
				if err := store.Delete(ctx, keyFunc(preData[0]), out, nil, nil); err != nil {
					t.Errorf("Expected no error when preparing watch case, %v", err)
				}
			},
			expectFn: func(g *GomegaWithT, events []watch.Event, err error) {
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(events).Should(HaveLen(2))
				g.Expect(events[0].Type).Should(Equal(watch.Added))
				g.Expect(events[1].Type).Should(Equal(watch.Deleted))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func testSetup(t *testing.T) (*store, *fake.Clientset, context.Context) {
	ns := "default"
	fakeCli := fake.NewSimpleClientset()
	codec := codecs.LegacyCodec(examplev1.SchemeGroupVersion)
	s, _, _ := NewApiServerStore(fakeCli, codec, ns, &example.Pod{}, func() runtime.Object { return &example.PodList{} })
	return s.(*store), fakeCli, context.Background()
}

func keyFunc(pod *example.Pod) string {
	ns := "default"
	if pod.Namespace != "" {
		ns = pod.Namespace
	}
	return fmt.Sprintf("/example/pods/%s/%s", ns, pod.Name)
}

func listKeyFunc(ns string) string {
	sb := strings.Builder{}
	sb.WriteString("/example/pods")
	if ns != "" {
		sb.WriteRune('/')
		sb.WriteString(ns)
	}
	return sb.String()
}

func predictionFunc(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: storage.DefaultNamespaceScopedAttr,
	}
}

func expectActions(g *GomegaWithT, actions []core.Action, lastN int, verb, resource string) {
	for i := 0; i < lastN; i++ {
		pos := len(actions) - 1 - i
		g.Expect(actions[pos].GetVerb()).Should(Equal(verb), "Expected the last %d action verb to be %s", verb)
		g.Expect(actions[pos].GetResource().Resource).Should(Equal(resource), "Expected the last %d action resource to be %s", resource)
	}
}
