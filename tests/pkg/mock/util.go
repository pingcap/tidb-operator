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

package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/calculate"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"k8s.io/klog"
)

type MonitorParams struct {
	KubernetesNamespace string   `json:"kubernetesNamespace"`
	Name                string   `json:"name"`
	MemberType          string   `json:"memberType"`
	Duration            string   `json:"duration"`
	Value               string   `json:"value"`
	InstancesPod        []string `json:"instances"`
	QueryType           string   `json:"queryType"`
	StorageType         string   `json:"storageType"`
}

type MonitorTargets struct {
	Status string             `json:"status"`
	Data   MonitorTargetsData `json:"data"`
}

type MonitorTargetsData struct {
	ActiveTargets []ActiveTargets `json:"activeTargets"`
}

type ActiveTargets struct {
	DiscoveredLabels DiscoveredLabels `json:"discoveredLabels"`
	Health           string           `json:"health"`
}

type DiscoveredLabels struct {
	Job     string `json:"job"`
	PodName string `json:"__meta_kubernetes_pod_name"`
}

func SetPrometheusResponse(monitorName, monitorNamespace string, mp *MonitorParams, fw portforward.PortForward) error {
	var prometheusAddr string
	if fw != nil {
		localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, monitorNamespace, fmt.Sprintf("svc/%s-prometheus", monitorName), 9090)
		if err != nil {
			return err
		}
		defer cancel()
		prometheusAddr = fmt.Sprintf("%s:%d", localHost, localPort)
	} else {
		prometheusAddr = fmt.Sprintf("%s-prometheus.%s:9090", monitorName, monitorNamespace)
	}
	b, err := json.Marshal(mp)
	if err != nil {
		return err
	}
	ep := fmt.Sprintf("http://%s/response", prometheusAddr)
	r, err := http.Post(ep, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	b, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if string(b) != "ok" {
		klog.Errorf("set mock-monitor response failed, response = %s", string(b))
		return fmt.Errorf("set mock-monitor response failed")
	}
	return nil
}

func buildPrometheusResponse(mp *MonitorParams) *calculate.Response {
	resp := &calculate.Response{}
	resp.Status = "success"
	resp.Data = calculate.Data{}
	cluster := mp.Name
	value := mp.Value
	if mp.QueryType == "cpu_usage" || mp.QueryType == "cpu_quota" {
		instances := mp.InstancesPod
		if instances == nil {
			return resp
		}
		for _, instance := range instances {
			r := calculate.Result{
				Metric: calculate.Metric{
					Instance:            instance,
					Cluster:             cluster,
					Job:                 mp.MemberType,
					KubernetesNamespace: mp.KubernetesNamespace,
					KubernetesNode:      "foo",
					KubernetesPodIp:     "foo",
				},
				Value: []interface{}{
					time.Now().UnixNano(),
					value,
				},
			}
			resp.Data.Result = append(resp.Data.Result, r)
		}
	}
	resp.Data.ResultType = "vector"
	return resp
}
