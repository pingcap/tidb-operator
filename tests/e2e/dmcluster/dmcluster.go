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

package dmcluster

import (
	"github.com/onsi/ginkgo"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = ginkgo.Describe("DMCluster", func() {
	f := e2eframework.NewDefaultFramework("dm-cluster")

	var (
		ns     string
		config *restclient.Config
		cli    versioned.Interface
	)

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		var err error
		config, err = framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for pingcap")
	})

	ginkgo.AfterEach(func() {})

	ginkgo.Context("[Feature:DM]", func() {
		ginkgo.It("setup replication for DM", func() {
			ginkgo.By("Deploy a basic dc")
			dc := fixture.GetDMCluster(ns, "basic-dm", utilimage.DMV2)
			dc.Spec.Master.Replicas = 1
			dc.Spec.Worker.Replicas = 1
			_, err := cli.PingcapV1alpha1().DMClusters(dc.Namespace).Create(dc)
			framework.ExpectNoError(err, "failed to create DmCluster: %q", dc.Name)
		})
	})
})
