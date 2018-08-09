// Copyright 2016 PingCAP, Inc.
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
	"path"

	"github.com/gorilla/mux"
	"github.com/pingcap/pd/server"
	"github.com/unrolled/render"
)

const pingAPI = "/ping"

func createRouter(prefix string, svr *server.Server) *mux.Router {
	rd := render.New(render.Options{
		IndentJSON: true,
	})

	router := mux.NewRouter().PathPrefix(prefix).Subrouter()
	handler := svr.GetHandler()

	operatorHandler := newOperatorHandler(handler, rd)
	router.HandleFunc("/api/v1/operators", operatorHandler.List).Methods("GET")
	router.HandleFunc("/api/v1/operators", operatorHandler.Post).Methods("POST")
	router.HandleFunc("/api/v1/operators/{region_id}", operatorHandler.Get).Methods("GET")
	router.HandleFunc("/api/v1/operators/{region_id}", operatorHandler.Delete).Methods("DELETE")

	schedulerHandler := newSchedulerHandler(handler, rd)
	router.HandleFunc("/api/v1/schedulers", schedulerHandler.List).Methods("GET")
	router.HandleFunc("/api/v1/schedulers", schedulerHandler.Post).Methods("POST")
	router.HandleFunc("/api/v1/schedulers/{name}", schedulerHandler.Delete).Methods("DELETE")

	router.Handle("/api/v1/cluster", newClusterHandler(svr, rd)).Methods("GET")
	router.HandleFunc("/api/v1/cluster/status", newClusterHandler(svr, rd).GetClusterStatus).Methods("GET")

	confHandler := newConfHandler(svr, rd)
	router.HandleFunc("/api/v1/config", confHandler.Get).Methods("GET")
	router.HandleFunc("/api/v1/config", confHandler.Post).Methods("POST")
	router.HandleFunc("/api/v1/config/schedule", confHandler.SetSchedule).Methods("POST")
	router.HandleFunc("/api/v1/config/schedule", confHandler.GetSchedule).Methods("GET")
	router.HandleFunc("/api/v1/config/replicate", confHandler.SetReplication).Methods("POST")
	router.HandleFunc("/api/v1/config/replicate", confHandler.GetReplication).Methods("GET")
	router.HandleFunc("/api/v1/config/namespace/{name}", confHandler.GetNamespace).Methods("GET")
	router.HandleFunc("/api/v1/config/namespace/{name}", confHandler.SetNamespace).Methods("POST")
	router.HandleFunc("/api/v1/config/namespace/{name}", confHandler.DeleteNamespace).Methods("DELETE")
	router.HandleFunc("/api/v1/config/label-property", confHandler.GetLabelProperty).Methods("GET")
	router.HandleFunc("/api/v1/config/label-property", confHandler.SetLabelProperty).Methods("POST")

	storeHandler := newStoreHandler(svr, rd)
	router.HandleFunc("/api/v1/store/{id}", storeHandler.Get).Methods("GET")
	router.HandleFunc("/api/v1/store/{id}", storeHandler.Delete).Methods("DELETE")
	router.HandleFunc("/api/v1/store/{id}/state", storeHandler.SetState).Methods("POST")
	router.HandleFunc("/api/v1/store/{id}/label", storeHandler.SetLabels).Methods("POST")
	router.HandleFunc("/api/v1/store/{id}/weight", storeHandler.SetWeight).Methods("POST")
	router.Handle("/api/v1/stores", newStoresHandler(svr, rd)).Methods("GET")

	labelsHandler := newLabelsHandler(svr, rd)
	router.HandleFunc("/api/v1/labels", labelsHandler.Get).Methods("GET")
	router.HandleFunc("/api/v1/labels/stores", labelsHandler.GetStores).Methods("GET")

	hotStatusHandler := newHotStatusHandler(handler, rd)
	router.HandleFunc("/api/v1/hotspot/regions/write", hotStatusHandler.GetHotWriteRegions).Methods("GET")
	router.HandleFunc("/api/v1/hotspot/regions/read", hotStatusHandler.GetHotReadRegions).Methods("GET")
	router.HandleFunc("/api/v1/hotspot/stores", hotStatusHandler.GetHotStores).Methods("GET")

	regionHandler := newRegionHandler(svr, rd)
	router.HandleFunc("/api/v1/region/id/{id}", regionHandler.GetRegionByID).Methods("GET")
	router.HandleFunc("/api/v1/region/key/{key}", regionHandler.GetRegionByKey).Methods("GET")

	regionsHandler := newRegionsHandler(svr, rd)
	router.HandleFunc("/api/v1/regions", regionsHandler.GetAll).Methods("GET")
	router.HandleFunc("/api/v1/regions/writeflow", regionsHandler.GetTopWriteFlow).Methods("GET")
	router.HandleFunc("/api/v1/regions/readflow", regionsHandler.GetTopReadFlow).Methods("GET")
	router.HandleFunc("/api/v1/regions/check/miss-peer", regionsHandler.GetMissPeerRegions).Methods("GET")
	router.HandleFunc("/api/v1/regions/check/extra-peer", regionsHandler.GetExtraPeerRegions).Methods("GET")
	router.HandleFunc("/api/v1/regions/check/pending-peer", regionsHandler.GetPendingPeerRegions).Methods("GET")
	router.HandleFunc("/api/v1/regions/check/down-peer", regionsHandler.GetDownPeerRegions).Methods("GET")
	router.HandleFunc("/api/v1/regions/sibling/{id}", regionsHandler.GetRegionSiblings).Methods("GET")
	router.HandleFunc("/api/v1/regions/check/incorrect-ns", regionsHandler.GetIncorrectNamespaceRegions).Methods("GET")

	router.Handle("/api/v1/version", newVersionHandler(rd)).Methods("GET")
	router.Handle("/api/v1/status", newStatusHandler(rd)).Methods("GET")

	memberHandler := newMemberHandler(svr, rd)
	router.HandleFunc("/api/v1/members", memberHandler.ListMembers).Methods("GET")
	router.HandleFunc("/api/v1/members/name/{name}", memberHandler.DeleteByName).Methods("DELETE")
	router.HandleFunc("/api/v1/members/id/{id}", memberHandler.DeleteByID).Methods("DELETE")
	router.HandleFunc("/api/v1/members/name/{name}", memberHandler.SetMemberPropertyByName).Methods("POST")

	leaderHandler := newLeaderHandler(svr, rd)
	router.HandleFunc("/api/v1/leader", leaderHandler.Get).Methods("GET")
	router.HandleFunc("/api/v1/leader/resign", leaderHandler.Resign).Methods("POST")
	router.HandleFunc("/api/v1/leader/transfer/{next_leader}", leaderHandler.Transfer).Methods("POST")

	classifierPrefix := path.Join(prefix, "/api/v1/classifier")
	classifierHandler := newClassifierHandler(svr, rd, classifierPrefix)
	router.PathPrefix("/api/v1/classifier/").Handler(classifierHandler)

	statsHandler := newStatsHandler(svr, rd)
	router.HandleFunc("/api/v1/stats/region", statsHandler.Region).Methods("GET")

	trendHandler := newTrendHandler(svr, rd)
	router.HandleFunc("/api/v1/trend", trendHandler.Handle).Methods("GET")

	adminHandler := newAdminHandler(svr, rd)
	router.HandleFunc("/api/v1/admin/cache/region/{id}", adminHandler.HandleDropCacheRegion).Methods("DELETE")

	logHanler := newlogHandler(svr, rd)
	router.HandleFunc("/api/v1/log", logHanler.Handle).Methods("POST")

	router.HandleFunc(pingAPI, func(w http.ResponseWriter, r *http.Request) {}).Methods("GET")
	router.Handle("/health", newHealthHandler(svr, rd)).Methods("GET")
	router.Handle("/diagnose", newDiagnoseHandler(svr, rd)).Methods("GET")
	return router
}
