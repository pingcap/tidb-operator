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
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	prometheusComponent = "prometheus"
	grafanaComponent    = "grafana"
	componentPrefix     = "/topology"
)

type TidbClusterStatusManager struct {
	cli             versioned.Interface
	pdControl       pdapi.PDControlInterface
	scalerLister    listers.TidbClusterAutoScalerLister
	tikvGroupLister listers.TiKVGroupLister
}

func NewTidbClusterStatusManager(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	scalerLister listers.TidbClusterAutoScalerLister,
	tikvGroupLister listers.TiKVGroupLister) *TidbClusterStatusManager {
	return &TidbClusterStatusManager{
		cli:             cli,
		pdControl:       pdapi.NewDefaultPDControl(kubeCli),
		scalerLister:    scalerLister,
		tikvGroupLister: tikvGroupLister,
	}
}

func (tcsm *TidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	return tcsm.syncTidbMonitorRefAndKey(tc)
}

func (tcsm *TidbClusterStatusManager) syncTidbMonitorRefAndKey(tc *v1alpha1.TidbCluster) error {
	tcsm.syncTikvGroupsStatus(tc)
	tm, err := tcsm.syncTidbMonitorRef(tc)
	if err != nil {
		return err
	}
	err = tcsm.syncDashboardMetricStorage(tc, tm)
	if err != nil {
		return err
	}
	return tcsm.syncAutoScalerRef(tc)
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
			err = nil
		}
		return nil, err
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
	tc.Status.Monitor.GrafanaEnabled = true
	if tm.Spec.Grafana == nil {
		tc.Status.Monitor.GrafanaEnabled = false
	}

	return tm, nil
}

func (tcsm *TidbClusterStatusManager) syncDashboardMetricStorage(tc *v1alpha1.TidbCluster, tm *v1alpha1.TidbMonitor) error {
	pdEtcdClient, err := tcsm.pdControl.GetPDEtcdClient(pdapi.Namespace(tc.Namespace), tc.Name, tc.Spec.PDAddress, tc.Spec.PD, tc.IsTLSClusterEnabled())
	if err != nil {
		return err
	}
	var prometheusExist bool
	var grafanaExist bool
	if tc.Status.Monitor != nil {
		prometheusExist = true
		grafanaExist = tc.Status.Monitor.GrafanaEnabled
	} else {
		prometheusExist = false
		grafanaExist = false
	}

	// sync prometheus key
	err = syncComponent(prometheusExist, tm, prometheusComponent, 9090, pdEtcdClient)
	if err != nil {
		return err
	}

	// sync grafana key
	err = syncComponent(grafanaExist, tm, grafanaComponent, 3000, pdEtcdClient)
	if err != nil {
		return err
	}
	return nil
}

func (tcsm *TidbClusterStatusManager) syncAutoScalerRef(tc *v1alpha1.TidbCluster) error {
	if tc.Status.AutoScaler == nil {
		klog.V(4).Infof("tc[%s/%s] autoscaler is empty", tc.Namespace, tc.Name)
		return nil
	}
	tacNamespace := tc.Status.AutoScaler.Namespace
	tacName := tc.Status.AutoScaler.Name
	tac, err := tcsm.scalerLister.TidbClusterAutoScalers(tacNamespace).Get(tacName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("tc[%s/%s] failed to find tac[%s/%s]", tc.Namespace, tc.Name, tacNamespace, tacName)
			tc.Status.AutoScaler = nil
			err = nil
		} else {
			err = fmt.Errorf("syncAutoScalerRef: failed to get tac %s/%s for cluster %s/%s, error: %s", tacNamespace, tacName, tc.GetNamespace(), tc.GetName(), err)
		}
		return err
	}
	if tac.Spec.Cluster.Name != tc.Name {
		klog.Infof("tc[%s/%s]'s target tac[%s/%s]'s cluster have been changed", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
		tc.Status.AutoScaler = nil
		return nil
	}
	if len(tac.Spec.Cluster.Namespace) < 1 {
		return nil
	}
	if tac.Spec.Cluster.Namespace != tc.Namespace {
		klog.Infof("tc[%s/%s]'s target tac[%s/%s]'s cluster namespace have been changed", tc.Namespace, tc.Name, tac.Namespace, tac.Name)
		tc.Status.AutoScaler = nil
		return nil
	}
	return nil
}

func (tcsm *TidbClusterStatusManager) syncTikvGroupsStatus(tc *v1alpha1.TidbCluster) {
	if tc.Status.TiKVGroups == nil || len(tc.Status.TiKVGroups) < 1 {
		return
	}

	var newGroups []v1alpha1.GroupRef
	for _, group := range tc.Status.TiKVGroups {
		tg, err := tcsm.tikvGroupLister.TiKVGroups(tc.Namespace).Get(group.Reference.Name)
		// If we failed to fetch the information for the registered tikvgroups, we will directly discard it.
		if err != nil {
			klog.Error(fmt.Errorf("syncTikvGroupsStatus: failed to get tikvgroups %s for cluster %s/%s, error: %s", group.Reference.Name, tc.GetNamespace(), tc.GetName(), err))
			continue
		}
		newGroups = append(newGroups, v1alpha1.GroupRef{Reference: corev1.LocalObjectReference{Name: tg.Name}})
	}
	tc.Status.TiKVGroups = newGroups
}

func syncComponent(exist bool, tm *v1alpha1.TidbMonitor, componentName string, port int, etcdClient pdapi.PDEtcdClient) error {
	key := buildComponentKey(componentName)
	if exist {
		v, err := buildComponentValue(tm, componentName, port)
		if err != nil {
			klog.Error(err.Error())
			return err
		}
		err = etcdClient.PutKey(key, v)
		if err != nil {
			klog.Error(err.Error())
			return err
		}
	} else {
		err := etcdClient.DeleteKey(key)
		if err != nil {
			klog.Error(err.Error())
			return err
		}
	}
	return nil
}

func buildComponentKey(component string) string {
	return fmt.Sprintf("%s/%s", componentPrefix, component)
}

func buildComponentValue(tm *v1alpha1.TidbMonitor, componentName string, port int) (string, error) {
	return buildEtcdValue(fmt.Sprintf("%s-%s.%s.svc", tm.Name, componentName, tm.Namespace), port)
}

type componentTopology struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func buildEtcdValue(host string, port int) (string, error) {
	topology := componentTopology{
		IP:   host,
		Port: port,
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
