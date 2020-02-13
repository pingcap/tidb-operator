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

package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type server struct {
	discovery discovery.TiDBDiscovery
}

// StartServer starts a TiDB Discovery server
func StartServer(cli versioned.Interface, kubeCli kubernetes.Interface, port int) {
	svr := &server{discovery.NewTiDBDiscovery(cli, kubeCli)}

	ws := new(restful.WebService)
	ws.Route(ws.GET("/new/{advertise-peer-url}").To(svr.newHandler))
	restful.Add(ws)

	klog.Infof("starting TiDB Discovery server, listening on 0.0.0.0:%d", port)
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func (svr *server) newHandler(req *restful.Request, resp *restful.Response) {
	encodedAdvertisePeerURL := req.PathParameter("advertise-peer-url")
	data, err := base64.StdEncoding.DecodeString(encodedAdvertisePeerURL)
	if err != nil {
		klog.Errorf("failed to decode advertise-peer-url: %s", encodedAdvertisePeerURL)
		if err := resp.WriteError(http.StatusInternalServerError, err); err != nil {
			klog.Errorf("failed to writeError: %v", err)
		}
		return
	}
	advertisePeerURL := string(data)

	result, err := svr.discovery.Discover(advertisePeerURL)
	if err != nil {
		klog.Errorf("failed to discover: %s, %v", advertisePeerURL, err)
		if err := resp.WriteError(http.StatusInternalServerError, err); err != nil {
			klog.Errorf("failed to writeError: %v", err)
		}
		return
	}

	klog.Infof("generated args for %s: %s", advertisePeerURL, result)
	if _, err := io.WriteString(resp, result); err != nil {
		klog.Errorf("failed to writeString: %s, %v", result, err)
	}
}
