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
	"fmt"
	"net/http"
	"sync"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/scheduler"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	schedulerapiv1 "k8s.io/kubernetes/pkg/scheduler/api/v1"
)

var (
	errFailToRead  = restful.NewError(http.StatusBadRequest, "unable to read request body")
	errFailToWrite = restful.NewError(http.StatusInternalServerError, "unable to write response")
)

type server struct {
	scheduler scheduler.Scheduler
	lock      sync.Mutex
}

// StartServer starts a kubernetes scheduler extender http apiserver
func StartServer(kubeCli kubernetes.Interface, cli versioned.Interface, port int) {
	s := scheduler.NewScheduler(kubeCli, cli)
	svr := &server{scheduler: s}

	ws := new(restful.WebService)
	ws.
		Path("/scheduler").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/filter").To(svr.filterNode).
		Doc("filter nodes").
		Operation("filterNodes").
		Writes(schedulerapiv1.ExtenderFilterResult{}))

	ws.Route(ws.POST("/preempt").To(svr.preemptNode).
		Doc("preempt nodes").
		Operation("preemptNodes").
		Writes(schedulerapiv1.ExtenderPreemptionResult{}))

	ws.Route(ws.POST("/prioritize").To(svr.prioritizeNode).
		Doc("prioritize nodes").
		Operation("prioritizeNodes").
		Writes(schedulerapiv1.HostPriorityList{}))
	restful.Add(ws)

	klog.Infof("start scheduler extender server, listening on 0.0.0.0:%d", port)
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func (svr *server) filterNode(req *restful.Request, resp *restful.Response) {
	svr.lock.Lock()
	defer svr.lock.Unlock()

	args := &schedulerapiv1.ExtenderArgs{}
	if err := req.ReadEntity(args); err != nil {
		errorResponse(resp, errFailToRead)
		return
	}

	filterResult, err := svr.scheduler.Filter(args)
	if err != nil {
		errorResponse(resp, restful.NewError(http.StatusInternalServerError,
			fmt.Sprintf("unable to filter nodes: %v", err)))
		return
	}

	if err := resp.WriteEntity(filterResult); err != nil {
		errorResponse(resp, errFailToWrite)
	}
}

func (svr *server) preemptNode(req *restful.Request, resp *restful.Response) {
	svr.lock.Lock()
	defer svr.lock.Unlock()

	args := &schedulerapiv1.ExtenderPreemptionArgs{}
	if err := req.ReadEntity(args); err != nil {
		errorResponse(resp, errFailToRead)
		return
	}

	preemptResult, err := svr.scheduler.Preempt(args)
	if err != nil {
		errorResponse(resp, restful.NewError(http.StatusInternalServerError,
			fmt.Sprintf("unable to preempt nodes: %v", err)))
		return
	}

	if err := resp.WriteEntity(preemptResult); err != nil {
		errorResponse(resp, errFailToWrite)
	}
}

func (svr *server) prioritizeNode(req *restful.Request, resp *restful.Response) {
	args := &schedulerapiv1.ExtenderArgs{}
	if err := req.ReadEntity(args); err != nil {
		errorResponse(resp, errFailToRead)
		return
	}

	priorityResult, err := svr.scheduler.Priority(args)
	if err != nil {
		errorResponse(resp, restful.NewError(http.StatusInternalServerError,
			fmt.Sprintf("unable to priority nodes: %v", err)))
		return
	}

	if err := resp.WriteEntity(priorityResult); err != nil {
		errorResponse(resp, errFailToWrite)
	}
}

func errorResponse(resp *restful.Response, svcErr restful.ServiceError) {
	klog.Error(svcErr.Message)
	if writeErr := resp.WriteServiceError(svcErr.Code, svcErr); writeErr != nil {
		klog.Errorf("unable to write error: %v", writeErr)
	}
}
