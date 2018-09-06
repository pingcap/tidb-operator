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
// limitations under the License.package spec

package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo" // revive:disable:dot-imports
	. "github.com/onsi/gomega" // revive:disable:dot-imports
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	cfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	cli, err = versioned.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	kubeCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	RunSpecs(t, "TiDB Operator Smoke tests")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	By("Clearing old TiDB Operator")
	Expect(clearOperator()).NotTo(HaveOccurred())

	By("Bootstrapping new TiDB Operator")
	Expect(installOperator()).NotTo(HaveOccurred())

	return nil
}, func(data []byte) {})

var _ = Describe("Smoke", func() {
	It("should create a tidb cluster", func() {
		testCreate()
	})
	It("should upgrade a tidb cluster", func() {
		testUpgrade()
	})
	It("should scale in/out a tidb cluster", func() {
		testScale()
	})

})
