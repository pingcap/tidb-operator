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
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"testing"
)

func TestSetTidbSpecDefault(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbCluster()
	setTidbSpecDefault(tc)
	g.Expect(tc.Spec.TiDB.Config).Should(BeNil())

	tc = newTidbCluster()
	tc.Spec.TiDB.Config = &v1alpha1.TiDBConfig{}
	setTidbSpecDefault(tc)
	g.Expect(*tc.Spec.TiDB.Config.Log.File.MaxBackups).Should(Equal(tidbLogMaxBackups))

	tc = newTidbCluster()
	oomAction := "cancel"
	tc.Spec.TiDB.Config = &v1alpha1.TiDBConfig{
		OOMAction: &oomAction,
	}
	setTidbSpecDefault(tc)
	g.Expect(*tc.Spec.TiDB.Config.Log.File.MaxBackups).Should(Equal(tidbLogMaxBackups))
	g.Expect(*tc.Spec.TiDB.Config.OOMAction).Should(Equal(oomAction))

	tc = newTidbCluster()
	infoLevel := "info"
	tc.Spec.TiDB.Config = &v1alpha1.TiDBConfig{
		OOMAction: &oomAction,
		Log: &v1alpha1.Log{
			Level: &infoLevel,
		},
	}
	setTidbSpecDefault(tc)
	g.Expect(*tc.Spec.TiDB.Config.Log.File.MaxBackups).Should(Equal(tidbLogMaxBackups))
	g.Expect(*tc.Spec.TiDB.Config.OOMAction).Should(Equal(oomAction))
	g.Expect(*tc.Spec.TiDB.Config.Log.Level).Should(Equal(infoLevel))

	tc = newTidbCluster()
	fileName := "slowlog.log"
	tc.Spec.TiDB.Config = &v1alpha1.TiDBConfig{
		OOMAction: &oomAction,
		Log: &v1alpha1.Log{
			Level: &infoLevel,
			File: &v1alpha1.FileLogConfig{
				Filename: &fileName,
			},
		},
	}
	setTidbSpecDefault(tc)
	g.Expect(*tc.Spec.TiDB.Config.Log.File.MaxBackups).Should(Equal(tidbLogMaxBackups))
	g.Expect(*tc.Spec.TiDB.Config.OOMAction).Should(Equal(oomAction))
	g.Expect(*tc.Spec.TiDB.Config.Log.Level).Should(Equal(infoLevel))
	g.Expect(*tc.Spec.TiDB.Config.Log.File.Filename).Should(Equal(fileName))

	tc = newTidbCluster()
	maxSize := 600
	tc.Spec.TiDB.Config = &v1alpha1.TiDBConfig{
		OOMAction: &oomAction,
		Log: &v1alpha1.Log{
			Level: &infoLevel,
			File: &v1alpha1.FileLogConfig{
				Filename: &fileName,
				MaxSize:  &maxSize,
			},
		},
	}
	setTidbSpecDefault(tc)
	g.Expect(*tc.Spec.TiDB.Config.Log.File.MaxSize).Should(Equal(maxSize))
	g.Expect(*tc.Spec.TiDB.Config.Log.File.MaxBackups).Should(Equal(tidbLogMaxBackups))
	g.Expect(*tc.Spec.TiDB.Config.OOMAction).Should(Equal(oomAction))
	g.Expect(*tc.Spec.TiDB.Config.Log.Level).Should(Equal(infoLevel))
	g.Expect(*tc.Spec.TiDB.Config.Log.File.Filename).Should(Equal(fileName))

}

func newTidbCluster() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		Spec: v1alpha1.TidbClusterSpec{
			PD:        &v1alpha1.PDSpec{},
			TiKV:      &v1alpha1.TiKVSpec{},
			TiDB:      &v1alpha1.TiDBSpec{},
			Discovery: &v1alpha1.DiscoverySpec{},
		},
	}
}
