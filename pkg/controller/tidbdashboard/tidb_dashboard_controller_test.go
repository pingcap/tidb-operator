// Copyright 2022 PingCAP, Inc.
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

package tidbdashboard

import (
	"fmt"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"

	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/cache"
)

func TestControllerSync(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name string

		addDashboardIndexer bool
		reconcile           func(dashboard *v1alpha1.TidbDashboard) error

		expectErrFn func(error)
	}

	cases := []testcase{
		{
			name:                "sync succeeded",
			addDashboardIndexer: true,
			reconcile:           nil,
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name:                "tidb dashboard isn't found",
			addDashboardIndexer: false,
			reconcile: func(td *v1alpha1.TidbDashboard) error {
				return fmt.Errorf("shouldn't arrive")
			},
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name: "reconcile tidb dashboard failed",
			reconcile: func(td *v1alpha1.TidbDashboard) error {
				return fmt.Errorf("reconcile failed")
			},
			addDashboardIndexer: true,
			expectErrFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err).Should(MatchError("reconcile failed"))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		fakeController, indexer := newFakeControllerForTest()
		control := fakeController.control.(*FakeTiDBDashboardControl)

		td := newTidbDashboardForTest()

		if testcase.reconcile != nil {
			control.MockReconcile(testcase.reconcile)
		}
		if testcase.addDashboardIndexer {
			err := indexer.Add(td)
			g.Expect(err).Should(Succeed())
		}

		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(td)
		g.Expect(err).Should(Succeed())

		err = fakeController.sync(key)
		testcase.expectErrFn(err)
	}
}

func newFakeControllerForTest() (*Controller, cache.Indexer) {
	fakeDeps := controller.NewFakeDependencies()
	indexer := fakeDeps.InformerFactory.Pingcap().V1alpha1().TidbDashboards().Informer().GetIndexer()
	control := &FakeTiDBDashboardControl{}

	fakeController := NewController(fakeDeps)
	fakeController.control = control

	return fakeController, indexer
}
