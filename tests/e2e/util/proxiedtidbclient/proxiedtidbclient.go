// Copyright 2019 PingCAP, Inc.
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

package proxiedtidbclient

import (
	"net/http"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
)

type proxiedTiDBClient struct {
	fw         portforward.PortForward
	httpClient *http.Client
	caCert     []byte
}

var _ controller.TiDBControlInterface = &proxiedTiDBClient{}

func (p *proxiedTiDBClient) GetHealth(tc *v1alpha1.TidbCluster, ordinal int32) (bool, error) {
	panic("implement when necessary")
}

func (p *proxiedTiDBClient) GetInfo(tc *v1alpha1.TidbCluster, ordinal int32) (*controller.DBInfo, error) {
	panic("implement when necessary")
}

func (p *proxiedTiDBClient) SetServerLabels(tc *v1alpha1.TidbCluster, ordinal int32, labels map[string]string) error {
	panic("implement when necessary")
}

func NewProxiedTiDBClient(fw portforward.PortForward, caCert []byte) controller.TiDBControlInterface {
	return &proxiedTiDBClient{fw: fw, httpClient: &http.Client{Timeout: 5 * time.Second}, caCert: caCert}
}
