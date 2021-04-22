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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
)

const (
	prometheusComponent = "prometheus"
	grafanaComponent    = "grafana"
	componentPrefix     = "/topology"
	tidbPrefix          = "/topology/tidb"

	tidbAddrPattern = `%s-tidb-\d+\.%s-tidb-peer\.%s\.svc%s`
)

type TidbClusterStatusManager struct {
	deps *controller.Dependencies
}

func NewTidbClusterStatusManager(deps *controller.Dependencies) *TidbClusterStatusManager {
	return &TidbClusterStatusManager{
		deps: deps,
	}
}

func (m *TidbClusterStatusManager) Sync(tc *v1alpha1.TidbCluster) error {
	err := m.syncTidbMonitorRefAndKey(tc)
	if err != nil {
		return err
	}

	return m.syncTiDBInfoKey(tc)
}

func (m *TidbClusterStatusManager) syncTidbMonitorRefAndKey(tc *v1alpha1.TidbCluster) error {
	tm, err := m.syncTidbMonitorRef(tc)
	if err != nil {
		return err
	}

	err = m.syncDashboardMetricStorage(tc, tm)
	if err != nil {
		return err
	}

	return m.syncAutoScalerRef(tc)
}

func (m *TidbClusterStatusManager) syncTidbMonitorRef(tc *v1alpha1.TidbCluster) (*v1alpha1.TidbMonitor, error) {
	if tc.Status.Monitor == nil {
		return nil, nil
	}

	tmRef := tc.Status.Monitor
	tm, err := m.deps.TiDBMonitorLister.TidbMonitors(tmRef.Namespace).Get(tmRef.Name)
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

func (m *TidbClusterStatusManager) syncDashboardMetricStorage(tc *v1alpha1.TidbCluster, tm *v1alpha1.TidbMonitor) error {
	if tc.Spec.PD == nil {
		return nil
	}
	pdEtcdClient, err := m.deps.PDControl.GetPDEtcdClient(pdapi.Namespace(tc.Namespace), tc.Name, tc.IsTLSClusterEnabled())
	if err != nil {
		return err
	}

	defer pdEtcdClient.Close()

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

// ref https://github.com/pingcap/tidb/blob/36b04d1aa01db722b3f07af759168c6b8da33801/domain/infosync/info.go#L72
// search `TopologyInformationPath` about how the key with 'ttl' and 'info' suffix is updated in that file.
func getStaleTidbInfoKey(ctx context.Context, client pdapi.PDEtcdClient) (staleKeys []*pdapi.KeyValue, err error) {
	kvs, err := client.Get(tidbPrefix, true /*prefix*/)
	if err != nil {
		return nil, perrors.AddStack(err)
	}

	infos := make(map[string]*pdapi.KeyValue)
	ttls := make(map[string]*pdapi.KeyValue)

	for _, kv := range kvs {
		key := kv.Key
		klog.V(4).Infof("Get TiDB info key %s", key)

		if strings.HasSuffix(key, "ttl") {
			ttls[key] = kv
		} else if strings.HasSuffix(key, "info") {
			infos[key] = kv
		}
	}

	for key, kv := range infos {
		if _, ok := ttls[strings.ReplaceAll(key, "/info", "/ttl")]; ok {
			continue
		}

		staleKeys = append(staleKeys, kv)
	}

	return
}

// /topology/tidb/basic-tidb-0.basic-tidb-peer.tidb-cluster.svc:4000/info -> basic-tidb-0.basic-tidb-peer.tidb-cluster.svc
func getTidbAddr(key string) (addr string) {
	key = strings.TrimPrefix(key, tidbPrefix)
	key = strings.TrimPrefix(key, "/")
	tmp := strings.Split(key, ":")

	addr = tmp[0]
	return
}

func (m *TidbClusterStatusManager) syncTiDBInfoKey(tc *v1alpha1.TidbCluster) error {
	var pdEtcdClient pdapi.PDEtcdClient
	var err error

	if tc.IsHeterogeneous() {
		pdEtcdClient, err = m.deps.PDControl.GetPDEtcdClient(pdapi.Namespace(tc.Namespace), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled())
	} else {
		pdEtcdClient, err = m.deps.PDControl.GetPDEtcdClient(pdapi.Namespace(tc.Namespace), tc.Name, tc.IsTLSClusterEnabled())
	}
	if err != nil {
		return err
	}

	defer pdEtcdClient.Close()

	kvs, err := getStaleTidbInfoKey(context.TODO(), pdEtcdClient)
	if err != nil {
		return err
	}

	expectAddrs := make(map[string]struct{})
	for ordinal := range tc.TiDBStsDesiredOrdinals(false) {
		addr := fmt.Sprintf("%s-%d.%s-tidb-peer.%s.svc",
			controller.TiDBMemberName(tc.GetName()),
			ordinal,
			tc.GetName(),
			tc.GetNamespace())
		if tc.Spec.ClusterDomain != "" {
			addr += "." + tc.Spec.ClusterDomain
		}

		expectAddrs[addr] = struct{}{}
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tidbAddrPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
	if err != nil {
		return err
	}

	for _, kv := range kvs {
		addr := getTidbAddr(kv.Key)
		// skip instance not own by this tc
		if !pattern.Match([]byte(addr)) {
			continue
		}

		if _, ok := expectAddrs[addr]; !ok {
			klog.V(2).Infof("Delete TiDB info key %s", kv.Key)
			err := pdEtcdClient.DeleteKey(kv.Key)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// syncAutoScalerRef delete the orphan info key that we do not expect the instance to be exist now logically.
func (m *TidbClusterStatusManager) syncAutoScalerRef(tc *v1alpha1.TidbCluster) error {
	if tc.Status.AutoScaler == nil {
		klog.V(4).Infof("tc[%s/%s] autoscaler is empty", tc.Namespace, tc.Name)
		return nil
	}
	tacNamespace := tc.Status.AutoScaler.Namespace
	tacName := tc.Status.AutoScaler.Name
	tac, err := m.deps.TiDBClusterAutoScalerLister.TidbClusterAutoScalers(tacNamespace).Get(tacName)
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
