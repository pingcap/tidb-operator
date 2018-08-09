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

package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pingcap/pd/server"
	"github.com/unrolled/render"
)

type adminHandler struct {
	svr *server.Server
	rd  *render.Render
}

func newAdminHandler(svr *server.Server, rd *render.Render) *adminHandler {
	return &adminHandler{
		svr: svr,
		rd:  rd,
	}
}

func (h *adminHandler) HandleDropCacheRegion(w http.ResponseWriter, r *http.Request) {
	cluster := h.svr.GetRaftCluster()
	if cluster == nil {
		h.rd.JSON(w, http.StatusInternalServerError, server.ErrNotBootstrapped.Error())
		return
	}

	vars := mux.Vars(r)
	regionIDStr := vars["id"]
	regionID, err := strconv.ParseUint(regionIDStr, 10, 64)
	if err != nil {
		h.rd.JSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	cluster.DropCacheRegion(regionID)
	h.rd.JSON(w, http.StatusOK, nil)
}
