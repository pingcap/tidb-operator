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

package member

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TidbClusterStatusManager struct {
	cli versioned.Interface
}

func NewTidbClusterStatusManager(cli versioned.Interface) *TidbClusterStatusManager {
	return &TidbClusterStatusManager{
		cli: cli,
	}
}

func (tcsm *TidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Status.Monitor != nil {
		tmRef := tc.Status.Monitor
		_, err := tcsm.cli.PingcapV1alpha1().TidbMonitors(tmRef.Namespace).Get(tmRef.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				tc.Status.Monitor = nil
			} else {
				return err
			}
		}
	}
	return nil
}

type FakeTidbClusterStatusManager struct {
}

func NewFakeTidbClusterStatusManager() *FakeTidbClusterStatusManager {
	return &FakeTidbClusterStatusManager{}
}

func (f *FakeTidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	return nil
}
