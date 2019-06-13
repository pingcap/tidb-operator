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

import (
	"fmt"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
)

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
	ws.Route(ws.POST("/vm/{name}/start").To(s.startVM))
	ws.Route(ws.POST("/vm/{name}/stop").To(s.stopVM))

	ws.Route(ws.POST(fmt.Sprintf("/%s/start", manager.ETCDService)).To(s.startETCD))
	ws.Route(ws.POST(fmt.Sprintf("/%s/stop", manager.ETCDService)).To(s.stopETCD))

	ws.Route(ws.POST(fmt.Sprintf("/%s/start", manager.KubeletService)).To(s.startKubelet))
	ws.Route(ws.POST(fmt.Sprintf("/%s/stop", manager.KubeletService)).To(s.stopKubelet))

	ws.Route(ws.POST(fmt.Sprintf("/%s/start", manager.KubeAPIServerService)).To(s.startKubeAPIServer))
	ws.Route(ws.POST(fmt.Sprintf("/%s/stop", manager.KubeAPIServerService)).To(s.stopKubeAPIServer))

	ws.Route(ws.POST(fmt.Sprintf("/%s/start", manager.KubeSchedulerService)).To(s.startKubeScheduler))
	ws.Route(ws.POST(fmt.Sprintf("/%s/stop", manager.KubeSchedulerService)).To(s.stopKubeScheduler))

	ws.Route(ws.POST(fmt.Sprintf("/%s/start", manager.KubeControllerManagerService)).To(s.startKubeControllerManager))
	ws.Route(ws.POST(fmt.Sprintf("/%s/stop", manager.KubeControllerManagerService)).To(s.stopKubeControllerManager))

	return ws
}
