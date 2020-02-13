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

package api

import (
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"k8s.io/klog"
)

// Server is a web service to control fault trigger
type Server struct {
	mgr *manager.Manager

	port int
}

// NewServer returns a api server
func NewServer(mgr *manager.Manager, port int) *Server {
	return &Server{
		mgr:  mgr,
		port: port,
	}
}

// StartServer starts a fault-trigger server
func (s *Server) StartServer() {
	ws := s.newService()

	restful.Add(ws)

	klog.Infof("starting fault-trigger server, listening on 0.0.0.0:%d", s.port)
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil))
}

func (s *Server) listVMs(req *restful.Request, resp *restful.Response) {
	res := newResponse("listVMs")
	vms, err := s.mgr.ListVMs()
	if err != nil {
		res.message(err.Error()).statusCode(http.StatusInternalServerError)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: listVMs, error: %v", err)
		}
		return
	}

	res.payload(vms).statusCode(http.StatusOK)

	if err = resp.WriteEntity(res); err != nil {
		klog.Errorf("failed to response, method: listVMs, error: %v", err)
	}
}

func (s *Server) startVM(req *restful.Request, resp *restful.Response) {
	res := newResponse("startVM")
	name := req.PathParameter("name")

	targetVM, err := s.getVM(name)
	if err != nil {
		res.message(fmt.Sprintf("failed to get vm %s, error: %v", name, err)).
			statusCode(http.StatusInternalServerError)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: startVM, error: %v", err)
		}
		return
	}

	if targetVM == nil {
		res.message(fmt.Sprintf("vm %s not found", name)).statusCode(http.StatusNotFound)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: startVM, error: %v", err)
		}
		return
	}

	s.vmAction(req, resp, res, targetVM, s.mgr.StartVM, "startVM")
}

func (s *Server) stopVM(req *restful.Request, resp *restful.Response) {
	res := newResponse("stopVM")
	name := req.PathParameter("name")

	targetVM, err := s.getVM(name)
	if err != nil {
		res.message(fmt.Sprintf("failed to get vm %s, error: %v", name, err)).
			statusCode(http.StatusInternalServerError)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: stopVM, error: %v", err)
		}
		return
	}

	if targetVM == nil {
		res.message(fmt.Sprintf("vm %s not found", name)).statusCode(http.StatusNotFound)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: stopVM, error: %v", err)
		}
		return
	}

	s.vmAction(req, resp, res, targetVM, s.mgr.StopVM, "stopVM")
}

func (s *Server) startETCD(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StartETCD, "startETCD")
}

func (s *Server) stopETCD(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StopETCD, "stopETCD")
}

func (s *Server) startKubelet(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StartKubelet, "startKubelet")
}

func (s *Server) stopKubelet(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StopKubelet, "stopKubelet")
}

func (s *Server) startKubeAPIServer(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StartKubeAPIServer, "startKubeAPIServer")
}

func (s *Server) stopKubeAPIServer(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StopKubeAPIServer, "stopKubeAPIServer")
}

func (s *Server) startKubeScheduler(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StartKubeScheduler, "startKubeScheduler")
}

func (s *Server) stopKubeScheduler(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StopKubeScheduler, "stopKubeScheduler")
}

func (s *Server) startKubeControllerManager(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StartKubeControllerManager, "startKubeControllerManager")
}

func (s *Server) stopKubeControllerManager(req *restful.Request, resp *restful.Response) {
	s.action(req, resp, s.mgr.StopKubeControllerManager, "stopKubeControllerManager")
}

func (s *Server) action(
	req *restful.Request,
	resp *restful.Response,
	fn func() error,
	method string,
) {
	res := newResponse(method)

	if err := fn(); err != nil {
		res.message(fmt.Sprintf("failed to %s, error: %v", method, err)).
			statusCode(http.StatusInternalServerError)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: %s, error: %v", method, err)
		}
		return
	}

	res.message("OK").statusCode(http.StatusOK)

	if err := resp.WriteEntity(res); err != nil {
		klog.Errorf("failed to response, method: %s, error: %v", method, err)
	}
}

func (s *Server) vmAction(
	req *restful.Request,
	resp *restful.Response,
	res *Response,
	targetVM *manager.VM,
	fn func(vm *manager.VM) error,
	method string,
) {
	if err := fn(targetVM); err != nil {
		res.message(fmt.Sprintf("failed to %s vm: %s, error: %v", method, targetVM.Name, err)).
			statusCode(http.StatusInternalServerError)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: %s, error: %v", method, err)
		}
		return
	}

	res.message("OK").statusCode(http.StatusOK)

	if err := resp.WriteEntity(res); err != nil {
		klog.Errorf("failed to response, method: %s, error: %v", method, err)
	}
}

func (s *Server) kubeProxyAction(
	req *restful.Request,
	resp *restful.Response,
	res *Response,
	nodeName string,
	fn func(nodeName string) error,
	method string,
) {
	if err := fn(nodeName); err != nil {
		res.message(fmt.Sprintf("failed to invoke %s, nodeName: %s, error: %v", method, nodeName, err)).
			statusCode(http.StatusInternalServerError)
		if err = resp.WriteEntity(res); err != nil {
			klog.Errorf("failed to response, methods: %s, error: %v", method, err)
		}
		return
	}

	res.message("OK").statusCode(http.StatusOK)

	if err := resp.WriteEntity(res); err != nil {
		klog.Errorf("failed to response, method: %s, error: %v", method, err)
	}
}

func (s *Server) getVM(name string) (*manager.VM, error) {
	vms, err := s.mgr.ListVMs()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		if name == vm.Name {
			return vm, nil
		}
	}

	return nil, nil
}
