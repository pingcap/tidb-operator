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

package defaulting

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func TestSetDMSpecDefault(t *testing.T) {
	g := NewGomegaWithT(t)

	dc := newDMCluster()
	SetDMClusterDefault(dc)
	g.Expect(dc.Spec.Master.Config).Should(BeNil())

	dc = newDMCluster()
	rpcTimeoutStr := "40s"
	dc.Spec.Master.Config = &v1alpha1.MasterConfig{
		RPCTimeoutStr: &rpcTimeoutStr,
	}
	SetDMClusterDefault(dc)
	g.Expect(*dc.Spec.Master.Config.RPCTimeoutStr).Should(Equal(rpcTimeoutStr))

	dc = newDMCluster()
	dc.Spec.Version = "v2.0.0-rc.2"
	keepAliveTTL := int64(15)
	dc.Spec.Worker.Config = &v1alpha1.WorkerConfig{
		KeepAliveTTL: &keepAliveTTL,
	}
	SetDMClusterDefault(dc)
	g.Expect(*dc.Spec.Worker.Config.KeepAliveTTL).Should(Equal(keepAliveTTL))
	g.Expect(*dc.Spec.Master.MaxFailoverCount).Should(Equal(int32(3)))
	g.Expect(dc.Spec.Master.BaseImage).Should(Equal(defaultMasterImage))
	g.Expect(*dc.Spec.Worker.MaxFailoverCount).Should(Equal(int32(3)))
	g.Expect(dc.Spec.Worker.BaseImage).Should(Equal(defaultWorkerImage))
}

func newDMCluster() *v1alpha1.DMCluster {
	return &v1alpha1.DMCluster{
		Spec: v1alpha1.DMClusterSpec{
			Master: v1alpha1.MasterSpec{},
			Worker: &v1alpha1.WorkerSpec{},
		},
	}
}
