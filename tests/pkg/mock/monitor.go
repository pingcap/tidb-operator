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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/autoscaler/autoscaler/calculate"
	"k8s.io/klog"
)

type MonitorInterface interface {
	ServeQuery(w http.ResponseWriter, r *http.Request)
	ServeTargets(w http.ResponseWriter, r *http.Request)
	SetResponse(w http.ResponseWriter, r *http.Request)
}

type mockPrometheus struct {
	// responses store the key from the query and value to answer the query
	// its not thread-safe, use it carefully
	responses map[string]string
}

func NewMockPrometheus() MonitorInterface {
	mp := &mockPrometheus{
		responses: map[string]string{},
	}
	upResp := BuildResponse(nil, "")
	b, err := json.Marshal(upResp)
	if err != nil {
		klog.Fatal(err.Error())
	}
	mp.responses["up"] = string(b)
	return mp
}

func (m *mockPrometheus) ServeQuery(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	key := ""
	for k, v := range params {
		if k == "query" && len(v) == 1 {
			key = v[0]
			break
		}
	}
	if len(key) < 1 {
		klog.Info()
		writeResponse(w, "no query param")
		return
	}
	klog.Infof("receive query, key: %s", key)
	v, ok := m.responses[key]
	if !ok {
		writeResponse(w, "no response value found")
		return
	}
	writeResponse(w, v)
}

func (m *mockPrometheus) SetResponse(w http.ResponseWriter, r *http.Request) {
	mp := &MonitorParams{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeResponse(w, err.Error())
		return
	}
	if err := json.Unmarshal(body, mp); err != nil {
		writeResponse(w, err.Error())
		return
	}

	b, err := json.Marshal(BuildResponse(mp.InstancesPod, mp.Value))
	if err != nil {
		writeResponse(w, err.Error())
		return
	}

	m.addIntoMaps(mp.Name, mp.MemberType, mp.Duration, string(b))
	writeResponse(w, "ok")
	return
}

func (m *mockPrometheus) ServeTargets(w http.ResponseWriter, r *http.Request) {
	data := &MonitorTargets{
		Status: "success",
		Data: MonitorTargetsData{
			ActiveTargets: []ActiveTargets{
				{
					DiscoveredLabels: DiscoveredLabels{
						Job:     "job",
						PodName: "pod",
					},
					Health: "true",
				},
			},
		},
	}
	b, err := json.Marshal(data)
	if err != nil {
		writeResponse(w, err.Error())
		return
	}
	writeResponse(w, string(b))
}

func (m *mockPrometheus) addIntoMaps(name, memberType, duration, value string) {
	key := ""
	klog.Infof("name= %s , memberType = %s , duration = %s , value = %s", name, memberType, duration, value)
	if memberType == "tidb" {
		key = fmt.Sprintf(calculate.TidbSumCpuMetricsPattern, name, duration)
	} else if memberType == "tikv" {
		key = fmt.Sprintf(calculate.TikvSumCpuMetricsPattern, name, duration)
	}
	m.responses[fmt.Sprintf("%s", key)] = value
	klog.Infof("add key: %s with value: %s", key, value)
}

func writeResponse(w http.ResponseWriter, msg string) {
	if _, err := w.Write([]byte(msg)); err != nil {
		klog.Error(err.Error())
	}
}
