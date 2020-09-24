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

const (
	TikvSumStorageMetricsPattern = `sum(tikv_store_size_bytes{cluster="%s", type="%s"}) by (cluster)`
	TikvSumCpuMetricsPattern     = `sum(increase(tikv_thread_cpu_seconds_total{cluster="%s"}[%s])) by (instance)`
	TidbSumCpuMetricsPattern     = `sum(increase(process_cpu_seconds_total{cluster="%s",job="tidb"}[%s])) by (instance)`
	InvalidTacMetricConfigureMsg = "tac[%s/%s] metric configuration invalid"
)

type SingleQuery struct {
	Endpoint  string
	Timestamp int64
	Query     string
	Instances []string
}
