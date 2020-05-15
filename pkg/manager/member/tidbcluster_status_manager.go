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
	"encoding/json"
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	prometheusEtcdKey = "/topology/prometheus"
)

type TidbClusterStatusManager struct {
	cli       versioned.Interface
	pdControl pdapi.PDControlInterface
}

func NewTidbClusterStatusManager(kubeCli kubernetes.Interface, cli versioned.Interface) *TidbClusterStatusManager {
	return &TidbClusterStatusManager{
		cli:       cli,
		pdControl: pdapi.NewDefaultPDControl(kubeCli),
	}
}

func (tcsm *TidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	if err := tcsm.syncTidbMonitorRefAndKey(tc); err != nil {
		return err
	}

	return nil
}

func (tcsm *TidbClusterStatusManager) syncTidbMonitorRefAndKey(tc *v1alpha1.TidbCluster) error {
	tm, err := tcsm.syncTidbMonitorRef(tc)
	if err != nil {
		return err
	}
	if err := tcsm.syncDashboardMetricStorage(tc, tm); err != nil {
		return err
	}
	return nil
}

func (tcsm *TidbClusterStatusManager) syncTidbMonitorRef(tc *v1alpha1.TidbCluster) (*v1alpha1.TidbMonitor, error) {
	if tc.Status.Monitor == nil {
		return nil, nil
	}
	tmRef := tc.Status.Monitor
	tm, err := tcsm.cli.PingcapV1alpha1().TidbMonitors(tmRef.Namespace).Get(tmRef.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			tc.Status.Monitor = nil
			return nil, nil
		} else {
			return nil, err
		}
	}
	tcRef := tm.Spec.Clusters
	if len(tcRef) < 1 {
		tc.Status.Monitor = nil
		return nil, nil
	}
	if len(tcRef[0].Namespace) < 1 {
		tcRef[0].Namespace = tm.Namespace
	}
	if tcRef[0].Name != tc.Name || tcRef[0].Namespace != tc.Namespace {
		tc.Status.Monitor = nil
		return nil, nil
	}

	return tm, nil
}

func (tcsm *TidbClusterStatusManager) syncDashboardMetricStorage(tc *v1alpha1.TidbCluster, tm *v1alpha1.TidbMonitor) error {
	pdEtcdClient, err := tcsm.pdControl.GetPDEtcdClient(pdapi.Namespace(tc.Namespace), tc.Name, tc.IsTLSClusterEnabled())
	if err != nil {
		return err
	}
	if tc.Status.Monitor != nil {
		tmRef := tc.Status.Monitor
		if tm == nil {
			err := fmt.Errorf("tc[%s/%s] have monitorRef while tm[%s/%s] get nil", tc.Namespace, tc.Name, tmRef.Namespace, tmRef.Name)
			klog.Errorf(err.Error())
			return err
		}
		v, err := buildPromethehusEtcdValue(tm)
		if err != nil {
			return err
		}
		err = pdEtcdClient.PutKey(prometheusEtcdKey, v)
		if err != nil {
			klog.Error(err.Error())
			return err
		}
		klog.Infof("tc[%s/%s]'s pd set tm[%s/%s]'s prometheus key", tc.Namespace, tc.Name, tm.Namespace, tm.Name)
		return nil
	}

	// If key doesn't exist, it would return nil
	err = pdEtcdClient.DeleteKey(prometheusEtcdKey)
	if err != nil {
		klog.Error(err.Error())
		return err
	}
	klog.Infof("tc[%s/%s]'s pd empty prometheus key", tc.Namespace, tc.Name)
	return nil
}

type componentTopology struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func buildPromethehusEtcdValue(tm *v1alpha1.TidbMonitor) (string, error) {
	topology := componentTopology{
		IP:   fmt.Sprintf("%s-prometheus.%s.svc", tm.Name, tm.Namespace),
		Port: 9090,
	}
	data, err := json.Marshal(topology)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type FakeTidbClusterStatusManager struct {
}

func NewFakeTidbClusterStatusManager() *FakeTidbClusterStatusManager {
	return &FakeTidbClusterStatusManager{}
}

func (f *FakeTidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	return nil
}
