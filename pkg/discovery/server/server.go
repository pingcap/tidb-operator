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

	"github.com/pingcap/tidb-operator/pkg/dmapi"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/discovery"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type server struct {
	discovery discovery.TiDBDiscovery
	container *restful.Container
}

// NewServer creates a new server.
func NewServer(pdControl pdapi.PDControlInterface, masterControl dmapi.MasterControlInterface, cli versioned.Interface, kubeCli kubernetes.Interface) Server {
	s := &server{
		discovery: discovery.NewTiDBDiscovery(pdControl, masterControl, cli, kubeCli),
		container: restful.NewContainer(),
	}
	s.registerHandlers()
	return s
}

func (s *server) registerHandlers() {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/new/{advertise-peer-url}").To(s.newHandler))
	ws.Route(ws.GET("/new/{advertise-peer-url}/{register-type}").To(s.newHandler))
	s.container.Add(ws)
}

func (s *server) ListenAndServe(addr string) {
	klog.Fatal(http.ListenAndServe(addr, s.container.ServeMux))
}

func (s *server) newHandler(req *restful.Request, resp *restful.Response) {
	encodedAdvertisePeerURL := req.PathParameter("advertise-peer-url")
	registerType := req.PathParameter("register-type")
	if registerType == "" {
		registerType = "pd"
	}
	data, err := base64.StdEncoding.DecodeString(encodedAdvertisePeerURL)
	if err != nil {
		klog.Errorf("failed to decode advertise-peer-url: %s, register-type is: %s", encodedAdvertisePeerURL, registerType)
		if err := resp.WriteError(http.StatusInternalServerError, err); err != nil {
			klog.Errorf("failed to writeError: %v", err)
		}
		return
	}
	advertisePeerURL := string(data)

	var result string
	switch registerType {
	case "pd":
		result, err = s.discovery.Discover(advertisePeerURL)
	case "dm":
		result, err = s.discovery.DiscoverDM(advertisePeerURL)
	default:
		klog.Errorf("invalid register-type %s", registerType)
		if err := resp.WriteError(http.StatusInternalServerError, fmt.Errorf("invalid register-type %s", registerType)); err != nil {
			klog.Errorf("failed to writeError: %v", err)
		}
		return
	}
	if err != nil {
		klog.Errorf("failed to discover: %s, %v, register-type is: %s", advertisePeerURL, err, registerType)
		if err := resp.WriteError(http.StatusInternalServerError, err); err != nil {
			klog.Errorf("failed to writeError: %v", err)
		}
		return
	}

	klog.Infof("generated args for %s: %s, register-type: %s", advertisePeerURL, result, registerType)
	if _, err := io.WriteString(resp, result); err != nil {
		klog.Errorf("failed to writeString: %s, %v", result, err)
	}
}
