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

package api

import restful "github.com/emicklei/go-restful"

const (
	// APIPrefix defines a prefix string for fault-trigger api
	APIPrefix = "/pingcap.com/api/v1"
)

func (s *Server) newService() *restful.WebService {
	ws := new(restful.WebService)
	ws.
		Path(APIPrefix).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/vms").To(s.listVMs))
	ws.Route(ws.GET("/vm/{name}/start").To(s.startVM))
	ws.Route(ws.GET("/vm/{name}/stop").To(s.stopVM))

	ws.Route(ws.GET("/etcd/start").To(s.startETCD))
	ws.Route(ws.GET("/etcd/stop").To(s.stopETCD))

	ws.Route(ws.GET("/kubelet/start").To(s.startKubelet))
	ws.Route(ws.GET("/kubelet/stop").To(s.stopKubelet))

	return ws
}
