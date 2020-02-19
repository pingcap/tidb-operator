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

package calculate

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	promClient "github.com/prometheus/client_golang/api"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	TikvSumCpuMetricsPattern     = `sum(tikv_thread_cpu_seconds_total{cluster="%s"}) by (instance)`
	TidbSumCpuMetricsPattern     = `sum(process_cpu_seconds_total{cluster="%s",job="tidb"}) by (instance)`
	InvalidTacMetricConfigureMsg = "tac[%s/%s] metric configuration invalid"
	CpuSumMetricsErrorMsg        = "tac[%s/%s] cpu sum metrics error, can't calculate the past %s cpu metrics, may caused by prometheus restart while data persistence not enabled"
	queryPath                    = "/api/v1/query"

	float64EqualityThreshold = 1e-9
)

func queryMetricsFromPrometheus(tac *v1alpha1.TidbClusterAutoScaler,
	query string, client promClient.Client, timestamp int64, resp *Response) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", *tac.Spec.MetricsUrl, queryPath), nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("query", query)
	q.Add("time", fmt.Sprintf("%d", timestamp))
	req.URL.RawQuery = q.Encode()
	r, body, err := client.Do(req.Context(), req)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("tac[%s/%s] query error, status code:%d", tac.Namespace, tac.Name, r.StatusCode)
	}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return err
	}
	if resp.Status != statusSuccess {
		return fmt.Errorf("tac[%s/%s] query error, response status: %v", tac.Namespace, tac.Name, resp.Status)
	}
	return nil
}

// sumByInstanceFromResponse sum the value in Response of each instance from Prometheus
func sumByInstanceFromResponse(instances []string, resp *Response) (float64, error) {
	s := sets.String{}
	for _, instance := range instances {
		s.Insert(instance)
	}
	sum := 0.0
	for _, r := range resp.Data.Result {
		if s.Has(r.Metric.Instance) {
			v, err := strconv.ParseFloat(r.Value[1].(string), 64)
			if err != nil {
				return 0.0, err
			}
			sum = sum + v
		}
	}
	return sum, nil
}

// calculate func calculate the recommended replicas by given usageRatio and currentReplicas
func calculate(currentValue float64, targetValue float64, currentReplicas int32) (int32, error) {
	if almostEqual(targetValue, 0.0) {
		return 0, fmt.Errorf("targetValue in calculate func can't be zero")
	}
	usageRatio := currentValue / targetValue
	return int32(math.Ceil(usageRatio * float64(currentReplicas))), nil
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}
