// Copyright 2021 PingCAP, Inc.
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

package tidbngmonitoring

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

		addTNGMToIndexer bool
		reconcile        func(*v1alpha1.TidbNGMonitoring) error

		expectErrFn func(error)
	}

	cases := []testcase{
		{
			name:             "sync succeeded",
			addTNGMToIndexer: true,
			reconcile:        nil,
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		{
			name:             "tidb ng monitoring isn't found",
			addTNGMToIndexer: false,
			reconcile: func(tngm *v1alpha1.TidbNGMonitoring) error {
				return fmt.Errorf("shouldn't arrive")
			},
			expectErrFn: func(err error) {
				g.Expect(err).Should(Succeed()) // should return nil when tngm isn't found
			},
		},
		{
			name: "reconcile tidb ng monitoring failed",
			reconcile: func(tngm *v1alpha1.TidbNGMonitoring) error {
				return fmt.Errorf("reconcile failed")
			},
			addTNGMToIndexer: true,
			expectErrFn: func(err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err).Should(MatchError("reconcile failed"))
			},
		},
	}

	for _, testcase := range cases {
		t.Logf("testcase: %s", testcase.name)

		controller, indexer := newFakeControllerForTest()
		control := controller.control.(*FakeTiDBNGMonitoringControl)

		tngm := newTidbNGMonitoringForTest()

		if testcase.reconcile != nil {
			control.MockReconcile(testcase.reconcile)
		}
		if testcase.addTNGMToIndexer {
			err := indexer.Add(tngm)
			g.Expect(err).Should(Succeed())
		}

		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(tngm)
		g.Expect(err).Should(Succeed())

		err = controller.sync(key)
		testcase.expectErrFn(err)
	}
}

func newFakeControllerForTest() (*Controller, cache.Indexer) {
	fakeDeps := controller.NewFakeDependencies()
	indexer := fakeDeps.InformerFactory.Pingcap().V1alpha1().TidbNGMonitorings().Informer().GetIndexer()
	control := &FakeTiDBNGMonitoringControl{}

	controller := NewController(fakeDeps)
	controller.control = control

	return controller, indexer
}
