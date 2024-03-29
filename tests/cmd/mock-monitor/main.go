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

package main

import (
	"net/http"

	"github.com/pingcap/tidb-operator/tests/pkg/mock"
	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
	"k8s.io/apiserver/pkg/server/healthz"
)

func main() {
	m := mock.NewMockPrometheus()
	http.HandleFunc("/api/v1/query", m.ServeQuery)
	http.HandleFunc("/response", m.SetResponse)
	http.HandleFunc("/api/v1/targets", m.ServeTargets)
	healthz.InstallHandler(http.DefaultServeMux)
	if err := http.ListenAndServe(":9090", nil); err != nil {
		log.Failf(err.Error())
	}
}
