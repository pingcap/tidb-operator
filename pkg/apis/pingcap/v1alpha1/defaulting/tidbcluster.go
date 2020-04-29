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

package defaulting

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	defaultTiDBImage    = "pingcap/tidb"
	defaultTiKVImage    = "pingcap/tikv"
	defaultPDImage      = "pingcap/pd"
	defaultBinlogImage  = "pingcap/tidb-binlog"
	defaultTiFlashImage = "pingcap/tiflash"
	defaultTiCdcImage   = "pingcap/ticdc"
)

var (
	tidbLogMaxBackups = 3
)

func SetTidbClusterDefault(tc *v1alpha1.TidbCluster) {
	setTidbClusterSpecDefault(tc)
	setPdSpecDefault(tc)
	setTikvSpecDefault(tc)
	setTidbSpecDefault(tc)
	if tc.Spec.Pump != nil {
		setPumpSpecDefault(tc)
	}
	if tc.Spec.TiFlash != nil {
		setTiFlashSpecDefault(tc)
	}
	if tc.Spec.TiCdc != nil {
		setTiCdcSpecDefault(tc)
	}
}

// setTidbClusterSpecDefault is only managed the property under Spec
func setTidbClusterSpecDefault(tc *v1alpha1.TidbCluster) {
	if string(tc.Spec.ImagePullPolicy) == "" {
		tc.Spec.ImagePullPolicy = corev1.PullIfNotPresent
	}
	if tc.Spec.TLSCluster == nil {
		tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: false}
	}
	if tc.Spec.EnablePVReclaim == nil {
		d := false
		tc.Spec.EnablePVReclaim = &d
	}
}

func setTidbSpecDefault(tc *v1alpha1.TidbCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.TiDB.Version != nil {
		if tc.Spec.TiDB.BaseImage == "" {
			tc.Spec.TiDB.BaseImage = defaultTiDBImage
		}
	}
	if tc.Spec.TiDB.MaxFailoverCount == nil {
		tc.Spec.TiDB.MaxFailoverCount = pointer.Int32Ptr(3)
	}

	// we only set default log
	if tc.Spec.TiDB.Config != nil {
		if tc.Spec.TiDB.Config.Log == nil {
			tc.Spec.TiDB.Config.Log = &v1alpha1.Log{
				File: &v1alpha1.FileLogConfig{
					MaxBackups: &tidbLogMaxBackups,
				},
			}
		} else {
			if tc.Spec.TiDB.Config.Log.File == nil {
				tc.Spec.TiDB.Config.Log.File = &v1alpha1.FileLogConfig{
					MaxBackups: &tidbLogMaxBackups,
				}
			} else {
				if tc.Spec.TiDB.Config.Log.File.MaxBackups == nil {
					tc.Spec.TiDB.Config.Log.File.MaxBackups = &tidbLogMaxBackups
				}
			}
		}
	}
}

func setTikvSpecDefault(tc *v1alpha1.TidbCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.TiKV.Version != nil {
		if tc.Spec.TiKV.BaseImage == "" {
			tc.Spec.TiKV.BaseImage = defaultTiKVImage
		}
	}
	if tc.Spec.TiKV.MaxFailoverCount == nil {
		tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(3)
	}
}

func setPdSpecDefault(tc *v1alpha1.TidbCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.PD.Version != nil {
		if tc.Spec.PD.BaseImage == "" {
			tc.Spec.PD.BaseImage = defaultPDImage
		}
	}
	if tc.Spec.PD.MaxFailoverCount == nil {
		tc.Spec.PD.MaxFailoverCount = pointer.Int32Ptr(3)
	}
}

func setPumpSpecDefault(tc *v1alpha1.TidbCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.Pump.Version != nil {
		if tc.Spec.Pump.BaseImage == "" {
			tc.Spec.Pump.BaseImage = defaultBinlogImage
		}
	}
}

func setTiFlashSpecDefault(tc *v1alpha1.TidbCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.TiFlash.Version != nil {
		if tc.Spec.TiFlash.BaseImage == "" {
			tc.Spec.TiFlash.BaseImage = defaultTiFlashImage
		}
	}
	if tc.Spec.TiFlash.MaxFailoverCount == nil {
		tc.Spec.TiFlash.MaxFailoverCount = pointer.Int32Ptr(3)
	}
}

func setTiCdcSpecDefault(tc *v1alpha1.TidbCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.TiCdc.Version != nil {
		if tc.Spec.TiCdc.BaseImage == "" {
			tc.Spec.TiCdc.BaseImage = defaultTiCdcImage
		}
	}
}
