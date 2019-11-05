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
	"strings"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/pkg/discovery"
	glog "k8s.io/klog"
)

type server struct {
	discovery discovery.TiDBDiscovery
}

// StartServer starts a TiDB Discovery server
// This will block until the http server fails
func StartServer(discover discovery.TiDBDiscovery, port int) error {
	svr := &server{discover}
	ws := new(restful.WebService)

	ws.Route(ws.GET("/cluster/{cluster-id}/addresses").To(svr.addresses))
	// use POST since this is side-effecting
	ws.Route(ws.POST("/cluster/{cluster-id}/pd/{pd-name}/{advertise-peer-url}").To(svr.idDiscovery))
	ws.Route(ws.DELETE("/cluster/{cluster-id}/pd/{pd-name}").To(svr.deletePD))
	// Original k8s url scheme
	ws.Route(ws.GET("/new/{advertise-peer-url}").To(svr.k8sDiscovery))

	restful.Add(ws)
	glog.Infof("starting TiDB Discovery server, listening on 0.0.0.0:%d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (svr *server) deletePD(req *restful.Request, resp *restful.Response) {
}
func (svr *server) addresses(req *restful.Request, resp *restful.Response) {
	clusterID := req.PathParameter("cluster-id")
	msg := fmt.Sprintf("get addresses clusterID=%s", clusterID)
	result, err := svr.discovery.GetClientAddresses(discovery.ClusterName(clusterID))
	if err != nil {
		respondErr(msg, resp, err)
		return
	}
	svr.sendResult(msg, resp, strings.Join(result, ","))
}

func (svr *server) k8sDiscovery(req *restful.Request, resp *restful.Response) {
	advertisePeerURL, err := getURL("advertise-peer-url", req)
	msg := advertisePeerURL
	if err != nil {
		respondErr(msg, resp, err)
		return
	}
	pdName, clusterID, url, err := discovery.ParseK8sAddress(advertisePeerURL)
	if err != nil {
		respondErr(msg, resp, err)
		return
	}
	result, err := svr.discovery.Discover(pdName, clusterID, url)
	if err != nil {
		respondErr(msg, resp, err)
		return
	}
	svr.sendResult(msg, resp, result)
}

func (svr *server) idDiscovery(req *restful.Request, resp *restful.Response) {
	clusterID := req.PathParameter("cluster-id")
	pdName := req.PathParameter("pd-name")
	advertisePeerURL, err := getURL("advertise-peer-url", req)
	msg := fmt.Sprintf("clusterID=%s pdName=%s url=%s", clusterID, pdName, advertisePeerURL)
	if err != nil {
		respondErr(msg, resp, err)
		return
	}
	url, err := discovery.ParseAddress(advertisePeerURL)
	if err != nil {
		respondErr(msg, resp, err)
		return
	}
	result, err := svr.discovery.Discover(discovery.PDName(pdName), discovery.ClusterName(clusterID), url)
	if err != nil {
		respondErr(msg, resp, err)
		return
	}
	svr.sendResult(msg, resp, result)
}

func respondErr(msg string, resp *restful.Response, err error) {
	glog.Errorf("failed to discover: %s, %v", msg, err)
	glog.Error(err)
	if writeErr := resp.WriteError(http.StatusBadRequest, err); writeErr != nil {
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

func (svr *server) sendResult(msg string, resp *restful.Response, result string) {
	glog.Infof("%s: %s", msg, result)
	if _, err := io.WriteString(resp, result); err != nil {
		glog.Errorf("failed to writeString: %s, %v", result, err)
	}
}
