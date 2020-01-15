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
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/scheduler"
	"k8s.io/client-go/kubernetes"
	glog "k8s.io/klog"
	schedulerapiv1 "k8s.io/kubernetes/pkg/scheduler/api/v1"
)

var (
	errFailToRead  = restful.NewError(http.StatusBadRequest, "unable to read request body")
	errFailToWrite = restful.NewError(http.StatusInternalServerError, "unable to write response")
	errTimeout     = restful.NewError(http.StatusRequestTimeout, "timeout to scheduler")
)

type server struct {
	scheduler           scheduler.Scheduler
	lockTimeoutDuration time.Duration
	lock                sync.Mutex
}

// StartServer starts a kubernetes scheduler extender http apiserver
func StartServer(kubeCli kubernetes.Interface, cli versioned.Interface, port int, lockTimeoutDuration time.Duration) {
	s := scheduler.NewScheduler(kubeCli, cli)
	svr := &server{
		scheduler:           s,
		lockTimeoutDuration: lockTimeoutDuration,
	}

	ws := new(restful.WebService)
	ws.
		Path("/scheduler").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/filter").To(svr.filterNode).
		Doc("filter nodes").
		Operation("filterNodes").
		Writes(schedulerapiv1.ExtenderFilterResult{}))

	ws.Route(ws.POST("/prioritize").To(svr.prioritizeNode).
		Doc("prioritize nodes").
		Operation("prioritizeNodes").
		Writes(schedulerapiv1.HostPriorityList{}))
	restful.Add(ws)

	glog.Infof("start scheduler extender server, listening on 0.0.0.0:%d", port)
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func (svr *server) filterNode(req *restful.Request, resp *restful.Response) {
	ch := make(chan *struct{})
	svr.lock.Lock()

	args := &schedulerapiv1.ExtenderArgs{}
	if err := req.ReadEntity(args); err != nil {
		errorResponse(resp, errFailToRead)
		return
	}

	var filterResult *schedulerapiv1.ExtenderFilterResult
	var err error

	go func() {
		filterResult, err = svr.scheduler.Filter(args)
		ch <- &struct{}{}
	}()

	select {
	case <-ch:
		if err != nil {
			errorResponse(resp, restful.NewError(http.StatusInternalServerError,
				fmt.Sprintf("unable to filter nodes: %v", err)))
			svr.lock.Unlock()
			return
		}

		if err := resp.WriteEntity(filterResult); err != nil {
			errorResponse(resp, errFailToWrite)
		}
		svr.lock.Unlock()
	case <-time.After(svr.lockTimeoutDuration):
		errorResponse(resp, errTimeout)
		svr.lock.Unlock()
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
	glog.Error(svcErr.Message)
	if writeErr := resp.WriteServiceError(svcErr.Code, svcErr); writeErr != nil {
		glog.Errorf("unable to write error: %v", writeErr)
	}
}
