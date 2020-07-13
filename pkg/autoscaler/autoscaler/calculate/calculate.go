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
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	promClient "github.com/prometheus/client_golang/api"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

const (
	TikvSumStorageMetricsPattern = `sum(tikv_store_size_bytes{cluster="%s", type="%s"}) by (cluster)`
	TikvSumCpuMetricsPattern     = `sum(increase(tikv_thread_cpu_seconds_total{cluster="%s"}[%s])) by (instance)`
	TidbSumCpuMetricsPattern     = `sum(increase(process_cpu_seconds_total{cluster="%s",job="tidb"}[%s])) by (instance)`
	InvalidTacMetricConfigureMsg = "tac[%s/%s] metric configuration invalid"
	queryPath                    = "/api/v1/query"

	float64EqualityThreshold = 1e-9
	httpRequestTimeout       = 5
)

type SingleQuery struct {
	Endpoint  string
	Timestamp int64
	Quary     string
	Instances []string
}

func queryMetricsFromPrometheus(tac *v1alpha1.TidbClusterAutoScaler, client promClient.Client, sq *SingleQuery, resp *Response) error {
	query := sq.Quary
	timestamp := sq.Timestamp

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*httpRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s%s", sq.Endpoint, queryPath), nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("query", query)
	q.Add("time", fmt.Sprintf("%d", timestamp))
	req.URL.RawQuery = q.Encode()
	r, body, err := client.Do(req.Context(), req)
	if err != nil {
		klog.Info(err.Error())
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

// sumForEachInstance sum the value in Response of each instance from Prometheus
func sumForEachInstance(instances []string, resp *Response) (float64, error) {
	if resp == nil {
		return 0, fmt.Errorf("metrics response from Promethus can't be empty")
	}
	s := sets.String{}
	for _, instance := range instances {
		s.Insert(instance)
	}
	sum := 0.0
	if len(resp.Data.Result) < 1 {
		return 0, fmt.Errorf("metrics Response return zero info")
	}

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
		return -1, fmt.Errorf("targetValue in calculate func can't be zero")
	}
	usageRatio := currentValue / targetValue
	return int32(math.Ceil(usageRatio * float64(currentReplicas))), nil
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}
