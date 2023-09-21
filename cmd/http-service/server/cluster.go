// Copyright 2023 PingCAP, Inc.
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
	"context"
	"errors"

	"github.com/pingcap/tidb-operator/http-service/pbgen/api"
)

type ClusterServer struct {
	api.UnimplementedClusterServer
}

func (s *ClusterServer) CreateCluster(context.Context, *api.CreateClusterReq) (*api.CreateClusterResp, error) {
	return nil, errors.New("not implemented")
}
