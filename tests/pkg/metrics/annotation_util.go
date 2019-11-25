// Copyright 2018 PingCAP, Inc.
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

package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/pingcap/tidb-operator/tests/slack"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricPort = 8090

//Client request grafana API on a set of resource paths.
type Client struct {
	// base is the root URL for all invocations of the client
	baseUrl url.URL
	client  *http.Client
}

//Annotation is a specification of the desired behavior of adding annotation
type Annotation struct {
	AnnotationOptions
	Text                string   `json:"text"`
	Tags                []string `json:"tags"`
	TimestampInMilliSec int64    `json:"time"`
}

//AnnotationOptions is the query options to a standard REST list call.
type AnnotationOptions struct {
	DashboardID int   `json:"dashboardId,omitempty"`
	PanelID     int   `json:"panelId,omitempty"`
	IsRegin     bool  `json:"isRegion,omitempty"`
	TimeEnd     int64 `json:"timeEnd,omitempty"`
}

//NewClient creates a new grafanaClient. This client performs rest functions
//such as Get, Post on specified paths.
func NewClient(grafanaURL string, userName string, password string) (*Client, error) {
	u, err := url.Parse(grafanaURL)
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(userName, password)
	return &Client{
		baseUrl: *u,
		client:  &http.Client{},
	}, nil
}

func (annotation Annotation) getBody() ([]byte, error) {
	body, err := json.Marshal(annotation)
	if err != nil {
		return nil, err
	}

	return body, nil
}

var (
	counterMetric     prometheus.Counter
	annotationSubPath = "api/annotations"
)

func initErrorMetric() prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "error_count",
		Help:        "record error count",
		ConstLabels: map[string]string{"fortest": "true"},
	})
}

//AddAnnotation adds an annotation to grafana.
func (cli *Client) AddAnnotation(annotation Annotation) error {
	body, err := annotation.getBody()
	if err != nil {
		return fmt.Errorf("create request body faield, %v", err)
	}

	req, err := http.NewRequest("POST", cli.getAnnotationPath(), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request failed, %v", err)
	}

	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	resp, err := cli.client.Do(req)
	if err != nil {
		return fmt.Errorf("add annotation faield, %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add annotation faield, statusCode=%v", resp.Status)
	}
	_, err = ioutil.ReadAll(resp.Body)
	return err
}

//IncrErrorCount increments the errorcount by 1.
func (cli *Client) IncrErrorCount() {
	counterMetric.Inc()
}

func (cli *Client) getAnnotationPath() string {
	u := cli.baseUrl
	u.Path = path.Join(cli.baseUrl.Path, annotationSubPath)
	return u.String()
}

func StartServer() {
	counterMetric = initErrorMetric()
	prometheus.MustRegister(counterMetric)
	mux := http.NewServeMux()

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", metricPort))
	if err != nil {
		fmt.Fprintf(os.Stderr, "listening port %d failed, %v", metricPort, err)
		slack.NotifyAndPanic(err)
	}

	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Handler: mux}
	go srv.Serve(l)
}
