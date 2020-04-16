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
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/calculate"
	"io/ioutil"
	"k8s.io/klog"
	"net/http"
)

type MonitorParams struct {
	Name       string `json:"name"`
	MemberType string `json:"type"`
	Duration   string `json:"duration"`
	Value      string `json:"value"`
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

func SetResponse(svc, name, duration, memberType, value string) error {
	ep := fmt.Sprintf("http://%s/response", svc)
	body := fmt.Sprintf(`{"name":"%s","type":"%s","duration":"%s","value":"%s"}`, name, memberType, duration, value)
	r, err := http.Post(ep, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if string(b) != "ok" {
		klog.Errorf("set mock-monitor response failed, response = %s", string(b))
		return fmt.Errorf("set mock-monitor response failed")
	}
	return nil
}

func BuildResponse(instances []string, value string) *calculate.Response {
	resp := &calculate.Response{}
	resp.Status = "success"
	resp.Data = calculate.Data{}
	if instances == nil {
		return resp
	}
	for _, instance := range instances {
		r := calculate.Result{
			Metric: calculate.Metric{
				Instance: instance,
			},
			Value: []interface{}{
				value,
				value,
			},
		}
		resp.Data.Result = append(resp.Data.Result, r)
	}
	return resp
}
