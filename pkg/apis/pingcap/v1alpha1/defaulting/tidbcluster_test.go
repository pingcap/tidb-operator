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

func TestSetTidbSpecDefault(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbCluster()
	setTidbSpecDefault(tc)
	g.Expect(tc.Spec.TiDB.Config).Should(BeNil())

	tc = newTidbCluster()
	tc.Spec.TiDB.Config = v1alpha1.NewTiDBConfig()
	setTidbSpecDefault(tc)
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.max-backups").AsInt()).Should(Equal(int64(tidbLogMaxBackups)))

	tc = newTidbCluster()
	tc.Spec.TiDB.Config = v1alpha1.NewTiDBConfig()
	oomAction := "cancel"
	tc.Spec.TiDB.Config.Set("oom-action", oomAction)
	setTidbSpecDefault(tc)
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.max-backups").AsInt()).Should(Equal(int64(tidbLogMaxBackups)))
	g.Expect(tc.Spec.TiDB.Config.Get("oom-action").AsString()).Should(Equal(oomAction))

	tc = newTidbCluster()
	tc.Spec.TiDB.Config = v1alpha1.NewTiDBConfig()
	infoLevel := "info"
	tc.Spec.TiDB.Config.Set("oom-action", oomAction)
	tc.Spec.TiDB.Config.Set("log.level", infoLevel)
	setTidbSpecDefault(tc)
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.max-backups").AsInt()).Should(Equal(int64(tidbLogMaxBackups)))
	g.Expect(tc.Spec.TiDB.Config.Get("oom-action").AsString()).Should(Equal(oomAction))
	g.Expect(tc.Spec.TiDB.Config.Get("log.level").AsString()).Should(Equal(infoLevel))

	tc = newTidbCluster()
	tc.Spec.TiDB.Config = v1alpha1.NewTiDBConfig()
	fileName := "slowlog.log"
	tc.Spec.TiDB.Config.Set("oom-action", oomAction)
	tc.Spec.TiDB.Config.Set("log.level", infoLevel)
	tc.Spec.TiDB.Config.Set("log.file.filename", fileName)
	setTidbSpecDefault(tc)
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.max-backups").AsInt()).Should(Equal(int64(tidbLogMaxBackups)))
	g.Expect(tc.Spec.TiDB.Config.Get("oom-action").AsString()).Should(Equal(oomAction))
	g.Expect(tc.Spec.TiDB.Config.Get("log.level").AsString()).Should(Equal(infoLevel))
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.filename").AsString()).Should(Equal(fileName))

	tc = newTidbCluster()
	tc.Spec.TiDB.Config = v1alpha1.NewTiDBConfig()
	var maxSize int64 = 600
	tc.Spec.TiDB.Config.Set("oom-action", oomAction)
	tc.Spec.TiDB.Config.Set("log.level", infoLevel)
	tc.Spec.TiDB.Config.Set("log.file.filename", fileName)
	tc.Spec.TiDB.Config.Set("log.file.max-size", maxSize)

	setTidbSpecDefault(tc)
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.max-backups").AsInt()).Should(Equal(int64(tidbLogMaxBackups)))
	g.Expect(tc.Spec.TiDB.Config.Get("oom-action").AsString()).Should(Equal(oomAction))
	g.Expect(tc.Spec.TiDB.Config.Get("log.level").AsString()).Should(Equal(infoLevel))
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.filename").AsString()).Should(Equal(fileName))
	g.Expect(tc.Spec.TiDB.Config.Get("log.file.max-size").AsInt()).Should(Equal(maxSize))

}

func newTidbCluster() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		Spec: v1alpha1.TidbClusterSpec{
			PD:   &v1alpha1.PDSpec{},
			TiKV: &v1alpha1.TiKVSpec{},
			TiDB: &v1alpha1.TiDBSpec{},
		},
	}
}
