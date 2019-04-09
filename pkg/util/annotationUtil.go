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

package util
import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
)

//Client request grafana API on a set of resource paths.
type grafanaCli struct {
	// base is the root URL for all invocations of the client
	baseUrl url.URL
	client *http.Client
}

//Annotation is a specification of the desired behavior of adding annotation
type Annotation struct {
	dashboardId int
	panelId int
	timestampInMilliSec int64
	tags []string
	text string
}

//NewGrafanaClient creats a new grafanaClient. This client performs rest functions
//such as Get, Post on specified paths.
func NewGrafanaClient(grafanaUrl string, userName string, password string) (*grafanaCli, error){
	u, err := url.Parse(grafanaUrl)
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(userName, password)

	return &grafanaCli{
		baseUrl: *u,
		client: &http.Client{},
	}, nil
}

func (annotation *Annotation) getBody() ([]byte, error) {
	m := map[string]interface{} {
		"dashboardId": annotation.dashboardId,
		"panelId": annotation.panelId,
		"time": annotation.timestampInMilliSec,
		"isRegion": false,
		"tags": annotation.tags,
		"text": annotation.text,
		"timeEnd": 0,
	}

	body, error := json.Marshal(m)
	if error != nil {
		return nil, error
	}

	return body, nil
}

var(
	initAction sync.Once
	counterMetric prometheus.Counter
	annotationSubPath = "api/annotations"
)

//initFunc is called with sync.Once, we use sync.Once to keep the thread safe.
func initFunc() {
	counterMetric = initErrorMetric()
	prometheus.MustRegister(counterMetric)
	mux := http.NewServeMux()

	l, err := net.Listen("tcp", ":8083")
	if err != nil {
		fmt.Fprint(os.Stderr, "listening port 8083 failed", err)
		panic(err)
	}

	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Handler: mux}
	go srv.Serve(l)
}

func initErrorMetric() prometheus.Counter{
	return prometheus.NewCounter(prometheus.CounterOpts{
		Name: "errorcount",
		Help: "record error count",
		ConstLabels: map[string]string{"fortest": "true"},
	})
}

//IncreErrorCountWithAnno increments the errorcount by 1,
//and add the annotation to grafanan.
func (cli *grafanaCli) IncreErrorCountWithAnno(annotation *Annotation) error{
	initAction.Do(initFunc)

	counterMetric.Inc()
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
	resp, error := cli.client.Do(req)
	if error != nil {
		return fmt.Errorf("add annotation faield, %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add annotation faield, statusCode=%v",  resp.Status)
	}

	return nil
}

func (cli *grafanaCli) getAnnotationPath() string {
	u := cli.baseUrl
	u.Path = path.Join(cli.baseUrl.Path, annotationSubPath)
	return u.String()
}




