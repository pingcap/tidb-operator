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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	httputil "github.com/pingcap/tidb-operator/pkg/util/http"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb/config"
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

func (p *proxiedTiDBClient) GetSettings(tc *v1alpha1.TidbCluster, ordinal int32) (*config.Config, error) {
	tcName := tc.GetName()
	ns := tc.GetNamespace()
	scheme := tc.Scheme()

	if tc.IsTLSClusterEnabled() {
		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(p.caCert)
		p.httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: rootCAs,
			},
		}
	}

	podName := fmt.Sprintf("%s-%d", controller.TiDBMemberName(tcName), ordinal)
	localHost, localPort, cancel, err := portforward.ForwardOnePort(p.fw, ns, fmt.Sprintf("pod/%s", podName), 10080)
	if err != nil {
		return nil, err
	}
	defer cancel()
	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", localHost, localPort),
		Path:   "/settings",
	}
	resp, err := p.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer httputil.DeferClose(resp.Body)
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf(fmt.Sprintf("Error response %v URL: %s", resp.StatusCode, u.String()))
		return nil, errMsg
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	info := config.Config{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func NewProxiedTiDBClient(fw portforward.PortForward, caCert []byte) controller.TiDBControlInterface {
	return &proxiedTiDBClient{fw: fw, httpClient: &http.Client{Timeout: 5 * time.Second}, caCert: caCert}
}
