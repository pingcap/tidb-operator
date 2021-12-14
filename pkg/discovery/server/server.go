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

	"github.com/pingcap/tidb-operator/pkg/dmapi"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/discovery"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
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
	ws.Route(ws.GET("/verify/{pd-url}").To(s.newVerifyHandler))
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
		if werr := resp.WriteError(http.StatusInternalServerError, err); werr != nil {
			klog.Errorf("failed to writeError: %v", werr)
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
		err = fmt.Errorf("invalid register-type %s", registerType)
		klog.Errorf("%v", err)
		if werr := resp.WriteError(http.StatusInternalServerError, err); werr != nil {
			klog.Errorf("failed to writeError: %v", werr)
		}
		return
	}
	if err != nil {
		klog.Errorf("failed to discover: %s, %v, register-type is: %s", advertisePeerURL, err, registerType)
		if werr := resp.WriteError(http.StatusInternalServerError, err); werr != nil {
			klog.Errorf("failed to writeError: %v", werr)
		}
		return
	}

	klog.Infof("generated args for %s: %s, register-type: %s", advertisePeerURL, result, registerType)
	if _, err := io.WriteString(resp, result); err != nil {
		klog.Errorf("failed to writeString: %s, %v", result, err)
	}

}

func (s *server) newVerifyHandler(req *restful.Request, resp *restful.Response) {
	encodedPDPeerURL := req.PathParameter("pd-url")
	data, err := base64.StdEncoding.DecodeString(encodedPDPeerURL)
	if err != nil {
		klog.Errorf("failed to decode pd-peer-url: %s", encodedPDPeerURL)
		if werr := resp.WriteError(http.StatusInternalServerError, err); werr != nil {
			klog.Errorf("failed to writeError: %v", werr)
		}
		return
	}
	pdPeerURL := string(data)
	pdPeerURL = strings.Trim(pdPeerURL, "\n")

	var result string
	result, err = s.discovery.VerifyPDEndpoint(pdPeerURL)
	if err != nil {
		klog.Errorf("failed to verify pd-url: %s, %v", pdPeerURL, err)
		if werr := resp.WriteError(http.StatusInternalServerError, err); werr != nil {
			klog.Errorf("failed to writeError: %v", werr)
		}
		// Return default value if verification failed
		result = pdPeerURL
	}

	klog.Infof("return pd-url for %s: %s", pdPeerURL, result)
	if _, err := io.WriteString(resp, result); err != nil {
		klog.Errorf("failed to writeString: %s, %v", result, err)
	}
}
