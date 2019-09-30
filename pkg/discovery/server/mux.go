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
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/discovery"
)

type server struct {
	discovery discovery.TiDBDiscovery
}

// StartServer starts a TiDB Discovery server
func StartServer(discover discovery.TiDBDiscovery, port int) {
	svr := &server{discover}
	ws := new(restful.WebService)

	// use POST since this is side-effecting
	ws.Route(ws.POST("/cluster/{cluster-id}/pd/{pd-name}/{advertise-peer-url}").To(svr.idDiscovery))
	// Original k8s url scheme
	ws.Route(ws.GET("/new/{advertise-peer-url}").To(svr.k8sDiscovery))

	restful.Add(ws)
	glog.Infof("starting TiDB Discovery server, listening on 0.0.0.0:%d", port)
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func (svr *server) k8sDiscovery(req *restful.Request, resp *restful.Response) {
	advertisePeerURL, err := getURL("advertise-peer-url", req)
	if err != nil {
		respondErr(resp, err)
		return
	}
	pdName, clusterID, url, err := discovery.ParseK8sURL(advertisePeerURL)
	if err != nil {
		respondErr(resp, err)
		return
	}
	result, err := svr.discovery.Discover(pdName, clusterID, url)
	svr.sendResult(advertisePeerURL, resp, result, err)
}

func (svr *server) idDiscovery(req *restful.Request, resp *restful.Response) {
	advertisePeerURL, err := getURL("advertise-peer-url", req)
	if err != nil {
		respondErr(resp, err)
		return
	}
	clusterID := req.PathParameter("cluster-id")
	pdName := req.PathParameter("pd-name")
	args := fmt.Sprint("clusterID=%s pdName=%s url=%s", clusterID, pdName, advertisePeerURL)
	url, err := discovery.ParseURL(advertisePeerURL)
	if err != nil {
		respondErr(resp, err)
		return
	}
	result, err := svr.discovery.Discover(discovery.PDName(pdName), discovery.ClusterID(clusterID), url)
	svr.sendResult(args, resp, result, err)
}

func respondErr(resp *restful.Response, err error) {
	glog.Error(err)
	if writeErr := resp.WriteError(http.StatusInternalServerError, err); writeErr != nil {
		glog.Errorf("failed to writeError: %v", writeErr)
	}
	return
}

func getURL(key string, req *restful.Request) (string, error) {
	encodedAdvertisePeerURL := req.PathParameter(key)
	data, err := base64.StdEncoding.DecodeString(encodedAdvertisePeerURL)
	if err != nil {
		return "", fmt.Errorf("input %s %+v", encodedAdvertisePeerURL, err)
	}
	return string(data), nil
}

func (svr *server) sendResult(args string, resp *restful.Response, result string, err error) {
	if err != nil {
		glog.Errorf("failed to discover: %s, %v", args, err)
		if err := resp.WriteError(http.StatusInternalServerError, err); err != nil {
			glog.Errorf("failed to writeError: %v", err)
		}
		return
	}

	glog.Infof("generated args for %s: %s", args, result)
	if _, err := io.WriteString(resp, result); err != nil {
		glog.Errorf("failed to writeString: %s, %v", result, err)
	}
}
