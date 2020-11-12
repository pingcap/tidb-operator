// Copyright 2020 PingCAP, Inc.
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
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFakeTidbCluster(t *testing.T) {
	g := NewGomegaWithT(t)

	deps := NewFakeDependencies()
	stop := make(chan struct{})
	defer close(stop)
	deps.InformerFactory.Start(stop)
	deps.InformerFactory.WaitForCacheSync(stop)

	for i := 0; i < 10; i++ {
		tc := &v1alpha1.TidbCluster{}
		tc.Namespace = "ns"
		tc.Name = "tcName" + strconv.Itoa(i)

		_, err := deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
		g.Expect(err).Should(BeNil())

		_, err = deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(tc.Name, v1.GetOptions{})
		g.Expect(err).Should(BeNil())

		g.Eventually(func() error {
			_, err := deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
			return err
		}, time.Second*10).Should(BeNil())
	}
}
